package stremio

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Dasio/go-stremio/pkg/cinemeta"
)

type customEndpoint struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// generateETag generates an ETag for the given data.
func generateETag(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// createManifestHandler creates a handler for manifest requests.
func createManifestHandler(manifest Manifest, logger *slog.Logger, callback ManifestCallback, userDataType reflect.Type, userDataIsBase64 bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user data from URL
		userData := r.PathValue("userData")
		if userData == "" {
			userData = r.URL.Query().Get("userData")
		}

		// Decode user data if needed
		decodedUserData, err := decodeUserData(userData, userDataType, logger, userDataIsBase64)
		if err != nil {
			logger.Error("Failed to decode user data", "error", err)
			http.Error(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		// Call callback if set
		if callback != nil {
			status := callback(r.Context(), &manifest, decodedUserData)
			if status >= 400 {
				http.Error(w, "Manifest callback returned error", status)
				return
			}
		}

		// Return manifest
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}
}

// createCatalogHandler creates a handler for catalog requests.
func createCatalogHandler(handlers map[string]CatalogHandler, cacheAge int, cachePublic bool, handleEtag bool, logger *slog.Logger, userDataType reflect.Type, userDataIsBase64 bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get type and ID from path parameters
		typeStr := r.PathValue("type")
		id := r.PathValue("id")
		if typeStr == "" || id == "" {
			http.Error(w, "Missing type or id parameter", http.StatusBadRequest)
			return
		}
		// Strip .json extension from id
		id = strings.TrimSuffix(id, ".json")

		// Get user data from URL
		userData := r.PathValue("userData")
		if userData == "" {
			userData = r.URL.Query().Get("userData")
		}

		// Decode user data if needed
		decodedUserData, err := decodeUserData(userData, userDataType, logger, userDataIsBase64)
		if err != nil {
			logger.Error("Failed to decode user data", "error", err)
			http.Error(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		// Get handler for type
		handler, ok := handlers[typeStr]
		if !ok {
			http.Error(w, "Unsupported type", http.StatusBadRequest)
			return
		}

		// Call handler
		items, err := handler(r.Context(), id, decodedUserData)
		if err != nil {
			logger.Error("Catalog handler returned error", "error", err)
			http.Error(w, "Failed to get catalog", http.StatusInternalServerError)
			return
		}

		// Set cache headers
		if cacheAge > 0 {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheAge))
		}
		if cachePublic {
			w.Header().Set("Cache-Control", "public")
		}

		// Handle ETag
		if handleEtag {
			etag := generateETag(items)
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Return items
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"metas": items})
	}
}

// create object fromt his

// createStreamHandler creates a handler for stream requests.
func createStreamHandler(handlers map[string]StreamHandler, cacheAge int, cachePublic bool, handleEtag bool, logger *slog.Logger, userDataType reflect.Type, userDataIsBase64 bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { // Get type and ID from path parameters
		typeStr := r.PathValue("type")
		id := r.PathValue("id")
		if typeStr == "" || id == "" {
			http.Error(w, "Missing type or id parameter", http.StatusBadRequest)
			return
		}
		// Strip .json extension from id
		id = strings.TrimSuffix(id, ".json")

		// Get user data from URL
		userData := r.PathValue("userData")
		if userData == "" {
			userData = r.URL.Query().Get("userData")
		}

		// Decode user data if needed
		decodedUserData, err := decodeUserData(userData, userDataType, logger, userDataIsBase64)
		if err != nil {
			logger.Error("Failed to decode user data", "error", err)
			http.Error(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		// Get handler for type
		handler, ok := handlers[typeStr]
		if !ok {
			http.Error(w, "Unsupported type", http.StatusBadRequest)
			return
		}

		// Call handler
		items, err := handler(r.Context(), id, decodedUserData)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			logger.Error("Stream handler returned error", "error", err)
			http.Error(w, "Failed to get streams", http.StatusInternalServerError)
			return
		}

		// Set cache headers
		if cacheAge > 0 {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheAge))
		}
		if cachePublic {
			w.Header().Set("Cache-Control", "public")
		}

		// Handle ETag
		if handleEtag {
			etag := generateETag(items)
			w.Header().Set("ETag", etag)
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Return items as {"streams": items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"streams": items})
	}
}

func createRootHandler(redirectURL string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("rootHandler called")
		logger.Info("Responding with redirect", "redirectURL", redirectURL)
		http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
	}
}

func decodeUserData(data string, t reflect.Type, logger *slog.Logger, userDataIsBase64 bool) (any, error) {
	if data == "" {
		// No user data provided, return empty string (as per SDK docs)
		return "", nil
	}

	if t == nil {
		// No user data type registered, return empty string (as per SDK docs)
		return "", nil
	}

	var userDataDecoded []byte
	var err error
	if userDataIsBase64 {
		// Remove padding so that both Base64URL values with and without padding work.
		data = strings.TrimSuffix(data, "=")
		userDataDecoded, err = base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(data)
	} else {
		var userDataDecodedString string
		userDataDecodedString, err = url.PathUnescape(data)
		userDataDecoded = []byte(userDataDecodedString)
	}
	if err != nil {
		// We use WARN instead of ERROR because it's most likely an *encoding* error on the client side
		logger.Warn("Couldn't decode user data", "error", err)
		return nil, err
	}

	userData := reflect.New(t).Interface()
	if err := json.Unmarshal(userDataDecoded, userData); err != nil {
		logger.Warn("Couldn't unmarshal user data", "error", err)
		return nil, err
	}
	logger.Info("Decoded user data", "userData", fmt.Sprintf("%+v", userData))
	return userData, nil
}

// createMetaMiddleware creates a middleware that fetches meta information for stream requests.
func createMetaMiddleware(metaClient MetaFetcher, putMetaInContext bool, logMediaName bool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get type and ID from path parameters
			typeStr := r.PathValue("type")
			id := r.PathValue("id")
			if typeStr == "" || id == "" {
				http.Error(w, "Missing type or id parameter", http.StatusBadRequest)
				return
			}
			// Strip .json extension from id
			id = strings.TrimSuffix(id, ".json")

			// Get meta information
			var meta cinemeta.Meta
			var err error
			if typeStr == "movie" {
				meta, err = metaClient.GetMovie(r.Context(), id)
			} else if typeStr == "series" {
				// Get season and episode from URL
				season := r.URL.Query().Get("season")
				episode := r.URL.Query().Get("episode")
				if season == "" || episode == "" {
					http.Error(w, "Missing season or episode parameter", http.StatusBadRequest)
					return
				}
				seasonNum, err := strconv.Atoi(season)
				if err != nil {
					http.Error(w, "Invalid season parameter", http.StatusBadRequest)
					return
				}
				episodeNum, err := strconv.Atoi(episode)
				if err != nil {
					http.Error(w, "Invalid episode parameter", http.StatusBadRequest)
					return
				}
				meta, err = metaClient.GetTVShow(r.Context(), id, seasonNum, episodeNum)
			} else {
				http.Error(w, "Unsupported type", http.StatusBadRequest)
				return
			}

			if err != nil {
				logger.Error("Failed to get meta information", "error", err)
				http.Error(w, "Failed to get meta information", http.StatusInternalServerError)
				return
			}

			// Log media name if enabled
			if logMediaName {
				logger.Info("Media name", "name", meta.Name)
			}

			// Put meta in context if enabled
			if putMetaInContext {
				ctx := context.WithValue(r.Context(), "meta", meta)
				r = r.WithContext(ctx)
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// addRouteMatcherMiddleware adds a middleware that matches routes and puts request info in the context.
func addRouteMatcherMiddleware(mux *http.ServeMux, configurationRequired bool, streamIDregex *regexp.Regexp, logger *slog.Logger) {
	// Create a middleware handler
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if configuration is required
			if configurationRequired {
				userData := r.PathValue("userData")
				if userData == "" {
					userData = r.URL.Query().Get("userData")
				}
				if userData == "" {
					http.Error(w, "Configuration required", http.StatusBadRequest)
					return
				}
			}

			// Check if stream ID matches regex
			if streamIDregex != nil {
				id := r.PathValue("id")
				if id == "" {
					id = r.URL.Query().Get("id")
				}
				if id != "" && !streamIDregex.MatchString(id) {
					http.Error(w, "Invalid stream ID", http.StatusBadRequest)
					return
				}
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}

	// Wrap the mux with the middleware
	originalMux := mux
	mux = http.NewServeMux()
	mux.Handle("/", middleware(originalMux))
}
