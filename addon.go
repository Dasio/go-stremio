package stremio

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/Dasio/go-stremio/pkg/cinemeta"
)

// ManifestCallback is the callback for manifest requests, so mostly addon installations.
// You can use the callback for two things:
//  1. To *prevent* users from installing your addon in Stremio.
//     The userData parameter depends on whether you called `RegisterUserData()` before:
//     If not, a simple string will be passed. It's empty if the user didn't provide user data.
//     If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
//     Return an HTTP status code >= 400 to stop further processing and let the addon return that exact status code.
//     Any status code < 400 will lead to the manifest being returned with a 200 OK status code in the response.
//  2. To *alter* the manifest before it's returned.
//     This can be useful for example if you want to return some catalogs depending on the userData.
//     Note that the manifest is only returned if the first return value is < 400 (see point 1.).
type ManifestCallback func(ctx context.Context, manifest *Manifest, userData any) int

// CatalogHandler is the callback for catalog requests for a specific type (like "movie").
// The id parameter is the catalog ID that you specified yourself in the CatalogItem objects in the Manifest.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
type CatalogHandler func(ctx context.Context, id string, userData any) ([]MetaPreviewItem, error)

// StreamHandler is the callback for stream requests for a specific type (like "movie").
// The context parameter contains a meta object under the key "meta" if PutMetaInContext was set to true in the addon options.
// The id parameter can be for example an IMDb ID if your addon handles the "movie" type.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
type StreamHandler func(ctx context.Context, id string, userData any) ([]StreamItem, error)

// ConfigurationHandler is the callback for configuration requests.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
type ConfigurationHandler func(ctx context.Context, userData any) (map[string]any, error)

// MetaFetcher returns metadata for movies and TV shows.
// It's used when you configure that the media name should be logged or that metadata should be put into the context.
type MetaFetcher interface {
	GetMovie(ctx context.Context, imdbID string) (cinemeta.Meta, error)
	GetTVShow(ctx context.Context, imdbID string, season int, episode int) (cinemeta.Meta, error)
}

// ConfigurationUI represents the configuration UI for the addon
type ConfigurationUI struct {
	Type       string         `json:"type"`               // The type of configuration UI (e.g. "form", "list", etc.)
	Properties map[string]any `json:"properties"`         // The properties of the configuration UI
	Required   []string       `json:"required,omitempty"` // The required fields
	Default    any            `json:"default,omitempty"`  // The default values
}

// Addon represents a remote addon.
// You can create one with NewAddon() and then run it with Run().
type Addon struct {
	manifest         Manifest
	catalogHandlers  map[string]CatalogHandler
	streamHandlers   map[string]StreamHandler
	configHandler    ConfigurationHandler
	configUI         *ConfigurationUI
	opts             Options
	logger           *slog.Logger
	customEndpoints  []customEndpoint
	manifestCallback ManifestCallback
	userDataType     reflect.Type
	metaClient       MetaFetcher
}

// NewAddon creates a new Addon object that can be started with Run().
// A proper manifest must be supplied, but manifestCallback and all but one handler can be nil in case you only want to handle specific requests and opts can be the zero value of Options.
func NewAddon(manifest Manifest, catalogHandlers map[string]CatalogHandler, streamHandlers map[string]StreamHandler, opts Options) (*Addon, error) {
	// Precondition checks
	if manifest.ID == "" || manifest.Name == "" || manifest.Description == "" || manifest.Version == "" {
		return nil, errors.New("An empty manifest was passed")
	} else if catalogHandlers == nil && streamHandlers == nil {
		return nil, errors.New("No handler was passed")
	} else if (opts.CachePublicCatalogs && opts.CacheAgeCatalogs == 0) ||
		(opts.CachePublicStreams && opts.CacheAgeStreams == 0) {
		return nil, errors.New("Enabling public caching only makes sense when also setting a cache age")
	} else if (opts.HandleEtagCatalogs && opts.CacheAgeCatalogs == 0) ||
		(opts.HandleEtagStreams && opts.CacheAgeStreams == 0) {
		return nil, errors.New("ETag handling only makes sense when also setting a cache age")
	} else if opts.DisableRequestLogging && (opts.LogIPs || opts.LogUserAgent) {
		return nil, errors.New("Enabling IP or user agent logging doesn't make sense when disabling request logging")
	} else if opts.Logger != nil && opts.LoggingLevel != "" {
		return nil, errors.New("Setting a logging level in the options doesn't make sense when you already set a custom logger")
	} else if opts.DisableRequestLogging && opts.LogMediaName {
		return nil, errors.New("Enabling media name logging doesn't make sense when disabling request logging")
	} else if opts.MetaClient != nil && !opts.LogMediaName && !opts.PutMetaInContext {
		return nil, errors.New("Setting a meta client when neither logging the media name nor putting it in the context doesn't make sense")
	} else if opts.MetaClient != nil && opts.CinemetaTimeout != 0 {
		return nil, errors.New("Setting a Cinemeta timeout doesn't make sense when you already set a meta client")
	} else if manifest.BehaviorHints.ConfigurationRequired && !manifest.BehaviorHints.Configurable {
		return nil, errors.New("Requiring a configuration only makes sense when also making the addon configurable")
	} else if opts.ConfigureHTMLfs != nil && !manifest.BehaviorHints.Configurable {
		return nil, errors.New("Setting a ConfigureHTMLfs only makes sense when also making the addon configurable")
	}

	// Set default values
	if opts.BindAddr == "" {
		opts.BindAddr = DefaultOptions.BindAddr
	}
	if opts.Port == 0 {
		opts.Port = DefaultOptions.Port
	}
	if opts.LoggingLevel == "" {
		opts.LoggingLevel = DefaultOptions.LoggingLevel
	}
	if opts.LogEncoding == "" {
		opts.LogEncoding = DefaultOptions.LogEncoding
	}
	if opts.CinemetaTimeout == 0 {
		opts.CinemetaTimeout = DefaultOptions.CinemetaTimeout
	}

	// Configure logger if no custom one is set
	if opts.Logger == nil {
		opts.Logger = NewLogger(opts.LoggingLevel, opts.LogEncoding)
	}

	// Configure Cinemeta client if no custom MetaFetcher is set
	if opts.MetaClient == nil && (opts.LogMediaName || opts.PutMetaInContext) {
		cinemetaCache := cinemeta.NewInMemoryCache()
		cinemetaOpts := cinemeta.ClientOptions{
			Timeout: opts.CinemetaTimeout,
		}
		opts.MetaClient = cinemeta.NewClient(cinemetaOpts, cinemetaCache, opts.Logger)
	}

	// Create and return addon
	return &Addon{
		manifest:        manifest,
		catalogHandlers: catalogHandlers,
		streamHandlers:  streamHandlers,
		opts:            opts,
		logger:          opts.Logger,
		metaClient:      opts.MetaClient,
	}, nil
}

// RegisterUserData registers the type of userData, so the addon can automatically unmarshal user data into an object of this type
// and pass the object into the manifest callback or catalog and stream handlers.
func (a *Addon) RegisterUserData(userDataObject any) {
	t := reflect.TypeOf(userDataObject)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	a.userDataType = t
}

// DecodeUserData decodes the request's user data and returns the result.
// It's useful when you add custom endpoints to the addon that don't have a userData parameter
// like the ManifestCallback, CatalogHandler and StreamHandler have.
// The param value must match the URL parameter you used when creating the custom endpoint,
// for example when using `AddEndpoint("GET", "/:userData/ping", customEndpoint)` you must pass "userData".
func (a *Addon) DecodeUserData(param string, r *http.Request) (any, error) {
	data := r.URL.Query().Get(param)
	return decodeUserData(data, a.userDataType, a.logger, a.opts.UserDataIsBase64)
}

// AddEndpoint adds a custom endpoint (a route and its handler).
// If you want to be able to access custom user data, you can use a path like this:
// "/:userData/foo" and then either deal with the data yourself
// by using `r.URL.Query().Get("userData")` in the handler,
// or use the convenience method `DecodeUserData("userData", r)`.
func (a *Addon) AddEndpoint(method, path string, handler http.HandlerFunc) {
	customEndpoint := customEndpoint{
		method:  method,
		path:    path,
		handler: handler,
	}
	a.customEndpoints = append(a.customEndpoints, customEndpoint)
}

// SetManifestCallback sets the manifest callback
func (a *Addon) SetManifestCallback(callback ManifestCallback) {
	a.manifestCallback = callback
}

// SetConfigurationHandler sets the configuration handler
func (a *Addon) SetConfigurationHandler(handler ConfigurationHandler) {
	a.configHandler = handler
}

// SetConfigurationUI sets the configuration UI for the addon
func (a *Addon) SetConfigurationUI(ui *ConfigurationUI) {
	a.configUI = ui
}

// Run starts the remote addon. It sets up an HTTP server that handles requests to "/manifest.json" etc. and gracefully handles shutdowns.
// The call is *blocking*, so use the stoppingChan param if you want to be notified when the addon is about to shut down
// because of a system signal like Ctrl+C or `docker stop`. It should be a buffered channel with a capacity of 1.
func (a *Addon) Run(stoppingChan chan bool) {
	logger := a.logger

	// Make sure the passed channel is buffered, so we can send a message before shutting down and not be blocked by the channel.
	if stoppingChan != nil && cap(stoppingChan) < 1 {
		logger.Error("The passed stopping channel isn't buffered")
		os.Exit(1)
	}

	// Create mux
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc("/health", createHealthHandler(logger))

	// Add manifest endpoint
	manifestHandler := createManifestHandler(a.manifest, logger, a.manifestCallback, a.userDataType, a.opts.UserDataIsBase64)
	mux.HandleFunc("/manifest.json", manifestHandler)
	mux.HandleFunc("/{userData}/manifest.json", manifestHandler)

	// Add catalog endpoint if handlers are set
	if a.catalogHandlers != nil {
		catalogHandler := createCatalogHandler(a.catalogHandlers, int(a.opts.CacheAgeCatalogs.Seconds()), a.opts.CachePublicCatalogs, a.opts.HandleEtagCatalogs, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			mux.HandleFunc("/catalog/{type}/{id}.json", catalogHandler)
		}
		mux.HandleFunc("/{userData}/catalog/{type}/{id}.json", catalogHandler)
	}

	// Add stream endpoint if handlers are set
	if a.streamHandlers != nil {
		streamHandler := createStreamHandler(a.streamHandlers, int(a.opts.CacheAgeStreams.Seconds()), a.opts.CachePublicStreams, a.opts.HandleEtagStreams, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			mux.HandleFunc("/stream/{type}/{id}.json", streamHandler)
		}
		mux.HandleFunc("/{userData}/stream/{type}/{id}.json", streamHandler)
	}

	// Add configuration endpoint if enabled
	if a.manifest.BehaviorHints.Configurable {
		if a.configHandler != nil {
			mux.HandleFunc("/configure", func(w http.ResponseWriter, r *http.Request) {
				userData, err := a.DecodeUserData("userData", r)
				if err != nil {
					http.Error(w, "Invalid user data", http.StatusBadRequest)
					return
				}

				config, err := a.configHandler(r.Context(), userData)
				if err != nil {
					http.Error(w, "Failed to get configuration", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(config)
			})
		}

		// Add configuration UI endpoint if set
		if a.configUI != nil {
			mux.HandleFunc("/configure.json", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(a.configUI)
			})
		}
	}

	// Add root redirect if configured
	if a.opts.RedirectURL != "" {
		mux.HandleFunc("/", createRootHandler(a.opts.RedirectURL, logger))
	}

	// Add custom endpoints
	for _, endpoint := range a.customEndpoints {
		mux.HandleFunc(endpoint.path, endpoint.handler)
	}

	// Add route matcher middleware
	var streamIDRegex *regexp.Regexp
	if a.opts.StreamIDregex != "" {
		var err error
		streamIDRegex, err = regexp.Compile(a.opts.StreamIDregex)
		if err != nil {
			logger.Error("Invalid stream ID regex", "error", err)
			os.Exit(1)
		}
	}
	addRouteMatcherMiddleware(mux, a.manifest.BehaviorHints.ConfigurationRequired, streamIDRegex, logger)

	// Add meta middleware if enabled
	if a.metaClient != nil {
		metaMw := createMetaMiddleware(a.metaClient, a.opts.PutMetaInContext, a.opts.LogMediaName, logger)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			mux.Handle("/stream/{type}/{id}.json", metaMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// This will be handled by the stream handler
				http.NotFound(w, r)
			})))
		}
		mux.Handle("/{userData}/stream/{type}/{id}.json", metaMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This will be handled by the stream handler
			http.NotFound(w, r)
		})))
	}

	// Create server
	server := &http.Server{
		Addr:    a.opts.BindAddr + ":" + strconv.Itoa(a.opts.Port),
		Handler: mux,
	}

	// Start server
	logger.Info("Starting server", "address", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Couldn't start server", "error", err)
			os.Exit(1)
		}
	}()

	// Handle shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	logger.Info("Received signal, shutting down server...", "signal", sig)

	if stoppingChan != nil {
		stoppingChan <- true
	}

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down server", "error", err)
		os.Exit(1)
	}

	logger.Info("Finished shutting down server")
}
