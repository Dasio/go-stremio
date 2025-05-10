package stremio

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dasio/go-stremio/pkg/cinemeta"
	"github.com/VictoriaMetrics/metrics"
)

type customMiddleware struct {
	path string
	mw   http.Handler
}

func createSlogLoggingMiddleware(logger *slog.Logger, logIPs, logUserAgent, logMediaName bool, requiresUserData bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer that captures status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			isStream := r.Context().Value("isStream") != nil

			// Get meta from context - the meta middleware put it there.
			var mediaName string
			if logMediaName && isStream {
				if meta, err := cinemeta.GetMetaFromContext(r.Context()); err != nil && err != cinemeta.ErrNoMeta {
					logger.Error("Couldn't get meta from context", "error", err)
				} else if err != cinemeta.ErrNoMeta {
					mediaName = fmt.Sprintf("%v (%v)", meta.Name, meta.ReleaseInfo)
				}
			}

			attrs := []any{
				"status", rw.statusCode,
				"duration", strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms",
				"method", r.Method,
				"url", r.URL.String(),
			}

			if logIPs {
				attrs = append(attrs,
					"ip", r.RemoteAddr,
					"forwardedFor", r.Header.Values("X-Forwarded-For"),
				)
			}
			if logUserAgent {
				attrs = append(attrs, "userAgent", r.UserAgent())
			}
			if logMediaName && isStream {
				if mediaName == "" {
					mediaName = "?"
				}
				attrs = append(attrs, "mediaName", mediaName)
			}

			logger.Info("Handled request", attrs...)
		})
	}
}

func createMetricsMiddleware() func(http.Handler) http.Handler {
	manifestRegex := regexp.MustCompile("^/.*/manifest.json$")
	catalogRegex := regexp.MustCompile(`^/.*/catalog/.*/.*\.json`)
	streamRegex := regexp.MustCompile(`^/.*/stream/.*/.*\.json`)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create response writer that captures status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			path := r.URL.Path
			var endpoint string
			switch path {
			case "/":
				endpoint = "root"
			case "/manifest.json":
				endpoint = "manifest"
			case "/configure":
				endpoint = "configure"
			case "/health":
				endpoint = "health"
			case "/metrics":
				endpoint = "metrics"
			}

			if endpoint == "" {
				if strings.HasPrefix(path, "/catalog") {
					endpoint = "catalog"
				} else if strings.HasPrefix(path, "/stream") {
					endpoint = "stream"
				} else if strings.HasPrefix(path, "/configure") {
					endpoint = "configure-other"
				} else if strings.HasPrefix(path, "/debug/pprof") {
					endpoint = "pprof"
				}
			}

			if endpoint == "" {
				if manifestRegex.MatchString(path) {
					endpoint = "manifest-data"
				} else if catalogRegex.MatchString(path) {
					endpoint = "catalog-data"
				} else if streamRegex.MatchString(path) {
					endpoint = "stream-data"
				}
			}

			// It would be valid for Prometheus to have an empty string as label, but it's confusing for users and makes custom legends in Grafana ugly.
			if endpoint == "" {
				endpoint = "other"
			}

			// Total number of HTTP requests.
			counterName := fmt.Sprintf(`http_requests_total{endpoint="%v", status="%v"}`, endpoint, rw.statusCode)
			counter := metrics.GetOrCreateCounter(counterName)
			counter.Add(1)
		})
	}
}

func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Origin, Accept-Encoding, Content-Language, X-Requested-With")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func createSlogMetaMiddleware(metaClient MetaFetcher, putMetaInHandlerContext, logMediaName bool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If we should put the meta in the context for *handlers* we get the meta synchronously.
			// Otherwise we only need it for logging and can get the meta asynchronously.
			if putMetaInHandlerContext {
				putMetaInContext(r, metaClient, logger)
				next.ServeHTTP(w, r)
			} else if logMediaName {
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					putMetaInContext(r, metaClient, logger)
					wg.Done()
				}()
				next.ServeHTTP(w, r)
				// Wait so that the meta is in the context when returning to the logging middleware
				wg.Wait()
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func putMetaInContext(r *http.Request, metaClient MetaFetcher, logger *slog.Logger) {
	var meta cinemeta.Meta
	var err error
	// type and id can never be empty, because that's been checked by a previous middleware
	t := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	id, err = url.PathUnescape(id)
	if err != nil {
		logger.Error("ID in URL parameters couldn't be unescaped", "id", id)
		return
	}

	switch t {
	case "movie":
		meta, err = metaClient.GetMovie(r.Context(), id)
		if err != nil {
			logger.Error("Couldn't get movie info with MetaFetcher", "error", err)
			return
		}
	case "series":
		splitID := strings.Split(id, ":")
		if len(splitID) != 3 {
			logger.Warn("No 3 elements after splitting TV show ID by \":\"", "id", id)
			return
		}
		season, err := strconv.Atoi(splitID[1])
		if err != nil {
			logger.Warn("Can't parse season as int", "season", splitID[1])
			return
		}
		episode, err := strconv.Atoi(splitID[2])
		if err != nil {
			logger.Warn("Can't parse episode as int", "episode", splitID[2])
			return
		}
		meta, err = metaClient.GetTVShow(r.Context(), splitID[0], season, episode)
		if err != nil {
			logger.Error("Couldn't get TV show info with MetaFetcher", "error", err)
			return
		}
	}

	logger.Debug("Got meta from cinemata client", "meta", fmt.Sprintf("%+v", meta))
	ctx := context.WithValue(r.Context(), "meta", meta)
	r = r.WithContext(ctx)
}
