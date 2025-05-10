package stremio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicAddon(t *testing.T) {
	// Create a basic manifest
	manifest := Manifest{
		ID:          "org.myexampleaddon",
		Version:     "1.0.0",
		Name:        "simple example",
		Description: "simple example",
		ResourceItems: []ResourceItem{
			{
				Name:  "stream",
				Types: []string{"movie"},
			},
		},
		Types:      []string{"movie"},
		IDprefixes: []string{"tt"},
	}

	// Create a stream handler that returns a stream for Big Buck Bunny
	streamHandlers := map[string]StreamHandler{
		"movie": func(ctx context.Context, id string, userData any) ([]StreamItem, error) {
			if id == "tt1254207" {
				return []StreamItem{
					{
						URL:   "http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_30fps_normal.mp4",
						Title: "1080p",
					},
				}, nil
			}
			return nil, NotFound
		},
	}

	// Create the addon
	addon, err := NewAddon(manifest, nil, streamHandlers, DefaultOptions)
	require.NoError(t, err)

	// Create a test server with the addon's handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", createManifestHandler(manifest, addon.logger, addon.manifestCallback, addon.userDataType, addon.opts.UserDataIsBase64))
	mux.HandleFunc("/stream", createStreamHandler(streamHandlers, int(addon.opts.CacheAgeStreams.Seconds()), addon.opts.CachePublicStreams, addon.opts.HandleEtagStreams, addon.logger, addon.userDataType, addon.opts.UserDataIsBase64))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test manifest endpoint
	t.Run("manifest", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/manifest.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var m Manifest
		err = json.NewDecoder(resp.Body).Decode(&m)
		require.NoError(t, err)
		require.Equal(t, manifest, m)
	})

	// Test stream endpoint for Big Buck Bunny
	t.Run("stream", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/stream?type=movie&id=tt1254207")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Streams []StreamItem `json:"streams"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.Len(t, result.Streams, 1)
		require.Equal(t, "http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_30fps_normal.mp4", result.Streams[0].URL)
		require.Equal(t, "1080p", result.Streams[0].Title)
	})

	// Test stream endpoint for non-existent movie
	t.Run("stream not found", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/stream?type=movie&id=tt0000000")
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
