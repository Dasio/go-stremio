package stremio

import (
	"log/slog"
	"net/http"
	"time"
)

// Options contains all options for creating a new addon.
type Options struct {
	// Server options
	BindAddr string
	Port     int

	// Logging options
	Logger                *slog.Logger
	LoggingLevel          string // "debug", "info", "warn", "error"
	LogEncoding           string // "json", "console"
	DisableRequestLogging bool
	LogIPs                bool
	LogUserAgent          bool
	LogMediaName          bool

	// Caching options
	CacheAgeCatalogs time.Duration
	CacheAgeStreams  time.Duration
	// If true, the addon will send Cache-Control headers with the max-age set to CacheAgeCatalogs/Streams.
	// This is useful when you have a CDN in front of your addon.
	CachePublicCatalogs bool
	CachePublicStreams  bool
	// If true, the addon will handle ETag headers for catalogs and streams.
	// This is useful when you have a CDN in front of your addon.
	HandleEtagCatalogs bool
	HandleEtagStreams  bool

	// Meta options
	MetaClient       MetaFetcher
	CinemetaTimeout  time.Duration
	PutMetaInContext bool

	// Configuration options
	ConfigureHTMLfs http.FileSystem

	// Other options
	Metrics     bool
	Profiling   bool
	RedirectURL string
	// If true, the addon will expect user data to be base64 encoded.
	UserDataIsBase64 bool
	// If set, the addon will only handle stream requests with IDs matching this regex.
	StreamIDregex string
}

// DefaultOptions contains the default values for Options.
var DefaultOptions = Options{
	BindAddr:     "0.0.0.0",
	Port:         8080,
	LoggingLevel: "info",
	LogEncoding:  "console",
}
