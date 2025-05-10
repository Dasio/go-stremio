package stremio

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
)

// SubtitleProvider represents a provider of subtitles
type SubtitleProvider interface {
	// GetSubtitles returns subtitles for a given video ID
	GetSubtitles(ctx context.Context, videoID string) ([]SubtitleItem, error)
}

// SubtitleHandler is the callback for subtitle requests
type SubtitleHandler func(ctx context.Context, videoID string, userData any) ([]SubtitleItem, error)

// createSubtitleHandler creates a handler for subtitle requests
func createSubtitleHandler(handler SubtitleHandler, cacheAge int, cachePublic bool, handleEtag bool, logger *slog.Logger, userDataType reflect.Type, userDataIsBase64 bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get video ID from URL
		videoID := r.URL.Query().Get("videoId")
		if videoID == "" {
			http.Error(w, "Missing videoId parameter", http.StatusBadRequest)
			return
		}

		// Get user data from URL
		userData, err := decodeUserData(r.URL.Query().Get("userData"), userDataType, logger, userDataIsBase64)
		if err != nil {
			http.Error(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		// Get subtitles from handler
		subtitles, err := handler(r.Context(), videoID, userData)
		if err != nil {
			logger.Error("Failed to get subtitles", "error", err)
			http.Error(w, "Failed to get subtitles", http.StatusInternalServerError)
			return
		}

		// Set cache headers
		if cacheAge > 0 {
			w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheAge))
		}

		// Return subtitles
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subtitles)
	}
}

// AddSubtitleHandler adds a subtitle handler to the addon
func (a *Addon) AddSubtitleHandler(handler SubtitleHandler) {
	a.customEndpoints = append(a.customEndpoints, customEndpoint{
		method:  "GET",
		path:    "/subtitles",
		handler: createSubtitleHandler(handler, int(a.opts.CacheAgeStreams.Seconds()), a.opts.CachePublicStreams, a.opts.HandleEtagStreams, a.logger, a.userDataType, a.opts.UserDataIsBase64),
	})
}
