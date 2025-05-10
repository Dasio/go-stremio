package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Dasio/go-stremio"
)

func main() {
	// Create a new addon
	addon, err := stremio.NewAddon(
		stremio.Manifest{
			ID:          "com.example.complete",
			Name:        "Complete Example",
			Description: "An example addon demonstrating all features",
			Version:     "1.0.0",
			ResourceItems: []stremio.ResourceItem{
				{
					Name:  "stream",
					Types: []string{"movie", "series"},
				},
				{
					Name:  "catalog",
					Types: []string{"movie", "series"},
				},
			},
			Types: []string{"movie", "series"},
			Catalogs: []stremio.CatalogItem{
				{
					Type: "movie",
					ID:   "movies",
					Name: "Movies",
				},
				{
					Type: "series",
					ID:   "series",
					Name: "TV Shows",
				},
			},
			IDprefixes: []string{"tt"},
			BehaviorHints: stremio.BehaviorHints{
				Configurable: true,
			},
		},
		map[string]stremio.CatalogHandler{
			"movie": func(ctx context.Context, id string, userData any) ([]stremio.MetaPreviewItem, error) {
				// Example catalog handler for movies
				return []stremio.MetaPreviewItem{
					{
						ID:          "tt0111161",
						Type:        "movie",
						Name:        "The Shawshank Redemption",
						Poster:      "https://example.com/poster.jpg",
						PosterShape: "poster",
						Links: []stremio.MetaLinkItem{
							{
								Name:     "Drama",
								Category: "genre",
								URL:      "https://example.com/genre/drama",
							},
						},
						IMDbRating:  "9.3",
						ReleaseInfo: "1994",
						Description: "Two imprisoned men bond over a number of years...",
					},
				}, nil
			},
		},
		map[string]stremio.StreamHandler{
			"movie": func(ctx context.Context, id string, userData any) ([]stremio.StreamItem, error) {
				// Example stream handler for movies
				return []stremio.StreamItem{
					{
						URL:   "https://example.com/movie.mp4",
						Title: "1080p",
						Subtitles: []stremio.SubtitleItem{
							{
								ID:       "en",
								URL:      "https://example.com/subtitles/en.srt",
								Language: "en",
								Label:    "English",
							},
						},
						BehaviorHints: stremio.StreamBehaviorHints{
							BingeWatch: true,
							AutoPlay:   true,
						},
					},
				}, nil
			},
		},
		stremio.Options{
			BindAddr:        "127.0.0.1",
			Port:            7000,
			LoggingLevel:    "info",
			LogEncoding:     "json",
			CacheAgeStreams: 24 * time.Hour,
		},
	)
	if err != nil {
		slog.Error("Failed to create addon", "error", err)
		os.Exit(1)
	}

	// Set up configuration UI
	configUI := stremio.NewConfigurationUI("form", map[string]any{
		"apiKey": stremio.NewConfigurationField("text", "API Key").
			SetDescription("Your API key for the service").
			SetRequired(true).
			SetPlaceholder("Enter your API key"),
		"quality": stremio.NewConfigurationField("select", "Stream Quality").
			SetDescription("Preferred stream quality").
			SetOptions([]any{"1080p", "720p", "480p"}).
			SetDefault("720p"),
	})
	addon.SetConfigurationUI(configUI)

	// Set up configuration handler
	addon.SetConfigurationHandler(func(ctx context.Context, userData any) (map[string]any, error) {
		// Example configuration handler
		return map[string]any{
			"apiKey":  "your-api-key",
			"quality": "720p",
		}, nil
	})

	// Set up subtitle handler
	addon.AddSubtitleHandler(func(ctx context.Context, videoID string, userData any) ([]stremio.SubtitleItem, error) {
		// Example subtitle handler
		return []stremio.SubtitleItem{
			{
				ID:       "en",
				URL:      "https://example.com/subtitles/en.srt",
				Language: "en",
				Label:    "English",
			},
			{
				ID:       "es",
				URL:      "https://example.com/subtitles/es.srt",
				Language: "es",
				Label:    "Spanish",
			},
		}, nil
	})

	// Run the addon
	stoppingChan := make(chan bool, 1)
	addon.Run(stoppingChan)
}
