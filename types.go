package stremio

// Manifest describes the capabilities of the addon.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/manifest.md
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// One of the following is required
	// Note: Can only have one in code because of how Go (de-)serialization works
	//Resources     []string       `json:"resources,omitempty"`
	ResourceItems []ResourceItem `json:"resources,omitempty"`

	Types    []string      `json:"types"` // Stremio supports "movie", "series", "channel" and "tv"
	Catalogs []CatalogItem `json:"catalogs"`

	// Optional
	IDprefixes    []string      `json:"idPrefixes,omitempty"`
	Background    string        `json:"background,omitempty"` // URL
	Logo          string        `json:"logo,omitempty"`       // URL
	ContactEmail  string        `json:"contactEmail,omitempty"`
	BehaviorHints BehaviorHints `json:"behaviorHints,omitempty"`
}

// clone returns a deep copy of m.
// We're not using one of the deep copy libraries because only few are maintained and even they have issues.
func (m Manifest) clone() Manifest {
	var resourceItems []ResourceItem
	if m.ResourceItems != nil {
		resourceItems = make([]ResourceItem, len(m.ResourceItems))
		for i, resourceItem := range m.ResourceItems {
			resourceItems[i] = resourceItem.clone()
		}
	}

	var types []string
	if m.Types != nil {
		types = make([]string, len(m.Types))
		for i, t := range m.Types {
			types[i] = t
		}
	}

	var catalogs []CatalogItem
	if m.Catalogs != nil {
		catalogs = make([]CatalogItem, len(m.Catalogs))
		for i, catalog := range m.Catalogs {
			catalogs[i] = catalog.clone()
		}
	}

	var idPrefixes []string
	if m.IDprefixes != nil {
		idPrefixes = make([]string, len(m.IDprefixes))
		for i, idPrefix := range m.IDprefixes {
			idPrefixes[i] = idPrefix
		}
	}

	return Manifest{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,

		ResourceItems: resourceItems,

		Types:    types,
		Catalogs: catalogs,

		IDprefixes:    idPrefixes,
		Background:    m.Background,
		Logo:          m.Logo,
		ContactEmail:  m.ContactEmail,
		BehaviorHints: m.BehaviorHints,
	}
}

type ResourceItem struct {
	Name  string   `json:"name"`
	Types []string `json:"types"` // Stremio supports "movie", "series", "channel" and "tv"

	// Optional
	IDprefixes []string `json:"idPrefixes,omitempty"`
}

func (ri ResourceItem) clone() ResourceItem {
	var types []string
	if ri.Types != nil {
		types = make([]string, len(ri.Types))
		for i, t := range ri.Types {
			types[i] = t
		}
	}

	var idPrefixes []string
	if ri.IDprefixes != nil {
		idPrefixes = make([]string, len(ri.IDprefixes))
		for i, idPrefix := range ri.IDprefixes {
			idPrefixes[i] = idPrefix
		}
	}

	return ResourceItem{
		Name:  ri.Name,
		Types: types,

		IDprefixes: idPrefixes,
	}
}

type BehaviorHints struct {
	// Note: Must include `omitempty`, otherwise it will be included if this struct is used in another one, even if the field of the containing struct is marked as `omitempty`
	Adult        bool `json:"adult,omitempty"`
	P2P          bool `json:"p2p,omitempty"`
	Configurable bool `json:"configurable,omitempty"`
	// If you set this to true, it will be true for the "/manifest.json" endpoint, but false for the "/:userData/manifest.json" endpoint, because otherwise Stremio won't show the "Install" button in its UI.
	ConfigurationRequired bool `json:"configurationRequired,omitempty"`
}

// CatalogItem represents a catalog.
type CatalogItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`

	// Optional
	Extra []ExtraItem `json:"extra,omitempty"`
}

func (ci CatalogItem) clone() CatalogItem {
	var extras []ExtraItem
	if ci.Extra != nil {
		extras = make([]ExtraItem, len(ci.Extra))
		for i, extra := range ci.Extra {
			extras[i] = extra.clone()
		}
	}

	return CatalogItem{
		Type: ci.Type,
		ID:   ci.ID,
		Name: ci.Name,

		Extra: extras,
	}
}

type ExtraItem struct {
	Name string `json:"name"`

	// Optional
	IsRequired   bool     `json:"isRequired,omitempty"`
	Options      []string `json:"options,omitempty"`
	OptionsLimit int      `json:"optionsLimit,omitempty"`
}

func (ei ExtraItem) clone() ExtraItem {
	var options []string
	if ei.Options != nil {
		options = make([]string, len(ei.Options))
		for i, option := range ei.Options {
			options[i] = option
		}
	}

	return ExtraItem{
		Name: ei.Name,

		IsRequired:   ei.IsRequired,
		Options:      options,
		OptionsLimit: ei.OptionsLimit,
	}
}

// MetaPreviewItem represents a meta preview item and is meant to be used within catalog responses.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.md#meta-preview-object
type MetaPreviewItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Poster string `json:"poster"` // URL

	// Optional
	PosterShape string `json:"posterShape,omitempty"`

	// Optional, used for the "Discover" page sidebar
	Genres        []string          `json:"genres,omitempty"`   // Will be replaced by Links at some point
	Director      []string          `json:"director,omitempty"` // Will be replaced by Links at some point
	Cast          []string          `json:"cast,omitempty"`     // Will be replaced by Links at some point
	Links         []MetaLinkItem    `json:"links,omitempty"`    // For genres, director, cast and potentially more
	IMDbRating    string            `json:"imdbRating,omitempty"`
	ReleaseInfo   string            `json:"releaseInfo,omitempty"` // E.g. "2000" for movies and "2000-2014" or "2000-" for TV shows
	Description   string            `json:"description,omitempty"`
	Released      string            `json:"released,omitempty"` // Must be ISO 8601, e.g. "2010-12-06T05:00:00.000Z"
	Runtime       string            `json:"runtime,omitempty"`
	Language      string            `json:"language,omitempty"`
	Country       string            `json:"country,omitempty"`
	Awards        string            `json:"awards,omitempty"`
	Website       string            `json:"website,omitempty"` // URL
	BehaviorHints MetaBehaviorHints `json:"behaviorHints,omitzero"`
}

// MetaItem represents a meta item and is meant to be used when info for a specific item was requested.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.md
type MetaItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`

	// Optional
	Genres           []string          `json:"genres,omitempty"`   // Will be replaced by Links at some point
	Director         []string          `json:"director,omitempty"` // Will be replaced by Links at some point
	Cast             []string          `json:"cast,omitempty"`     // Will be replaced by Links at some point
	Links            []MetaLinkItem    `json:"links,omitempty"`    // For genres, director, cast and potentially more
	Poster           string            `json:"poster,omitempty"`   // URL
	PosterShape      string            `json:"posterShape,omitempty"`
	Background       string            `json:"background,omitempty"` // URL
	Logo             string            `json:"logo,omitempty"`       // URL
	Description      string            `json:"description,omitempty"`
	ReleaseInfo      string            `json:"releaseInfo,omitempty"` // E.g. "2000" for movies and "2000-2014" or "2000-" for TV shows
	IMDbRating       string            `json:"imdbRating,omitempty"`
	Released         string            `json:"released,omitempty"` // Must be ISO 8601, e.g. "2010-12-06T05:00:00.000Z"
	Videos           []VideoItem       `json:"videos,omitempty"`
	Runtime          string            `json:"runtime,omitempty"`
	Language         string            `json:"language,omitempty"`
	Country          string            `json:"country,omitempty"`
	Awards           string            `json:"awards,omitempty"`
	Website          string            `json:"website,omitempty"` // URL
	BehaviorHints    MetaBehaviorHints `json:"behaviorHints,omitempty"`
	Popularity       float64           `json:"popularity,omitempty"`
	VoteAverage      float64           `json:"voteAverage,omitempty"`
	VoteCount        int               `json:"voteCount,omitempty"`
	Status           string            `json:"status,omitempty"`           // For TV shows: "Returning Series", "Ended", etc.
	Network          string            `json:"network,omitempty"`          // For TV shows
	EpisodeRunTime   []int             `json:"episodeRunTime,omitempty"`   // For TV shows
	InProduction     bool              `json:"inProduction,omitempty"`     // For TV shows
	LastAirDate      string            `json:"lastAirDate,omitempty"`      // For TV shows
	NumberOfSeasons  int               `json:"numberOfSeasons,omitempty"`  // For TV shows
	NumberOfEpisodes int               `json:"numberOfEpisodes,omitempty"` // For TV shows
	CreatedBy        []string          `json:"createdBy,omitempty"`        // For TV shows
	Seasons          []SeasonItem      `json:"seasons,omitempty"`          // For TV shows
}

// SeasonItem represents a season in a TV show
type SeasonItem struct {
	ID           string `json:"id"`
	Season       int    `json:"season"`
	Name         string `json:"name"`
	Overview     string `json:"overview,omitempty"`
	Poster       string `json:"poster,omitempty"`  // URL
	AirDate      string `json:"airDate,omitempty"` // Must be ISO 8601
	EpisodeCount int    `json:"episodeCount,omitempty"`
}

// MetaBehaviorHints provides additional information about the meta item
type MetaBehaviorHints struct {
	DefaultVideoID string `json:"defaultVideoId,omitempty"` // ID of the default video to play
	DefaultSeason  string `json:"defaultSeason,omitempty"`  // Default season for TV shows
	DefaultEpisode string `json:"defaultEpisode,omitempty"` // Default episode for TV shows
	AutoPlay       bool   `json:"autoPlay,omitempty"`       // Whether to auto-play the content
	BingeWatch     bool   `json:"bingeWatch,omitempty"`     // Whether the content supports binge watching
	Proxy          bool   `json:"proxy,omitempty"`          // Whether the content should be proxied
}

// MetaLinkItem links to a page within Stremio.
// It will at some point replace the usage of `genres`, `director` and `cast`.
// Note: It's not fully supported by Stremio yet (not fully on PC and not at all on Android)!
type MetaLinkItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"` //  // URL. Can be "Meta Links" (see https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.links.md)
}

type VideoItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Released string `json:"released"` // Must be ISO 8601, e.g. "2010-12-06T05:00:00.000Z"

	// Optional
	Thumbnail string       `json:"thumbnail,omitempty"` // URL
	Streams   []StreamItem `json:"streams,omitempty"`
	Available bool         `json:"available,omitempty"`
	Episode   string       `json:"episode,omitempty"`
	Season    string       `json:"season,omitempty"`
	Trailer   string       `json:"trailer,omitempty"` // Youtube ID
	Overview  string       `json:"overview,omitempty"`
}

// StreamItem represents a stream for a MetaItem.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/stream.md
type StreamItem struct {
	// One of the following is required
	URL         string `json:"url,omitempty"`         // Direct URL to a video stream - must be an MP4 through https; others supported (other video formats over http/rtmp supported if notWebReady is set)
	YoutubeID   string `json:"ytId,omitempty"`        // YouTube video ID, plays using the built-in YouTube player
	InfoHash    string `json:"infoHash,omitempty"`    // Info hash of a torrent file
	ExternalURL string `json:"externalUrl,omitempty"` // Meta Link or an external URL to the video, which should be opened in a browser

	// Optional
	Name          string              `json:"name,omitempty"`        // Name of the stream; usually used for stream quality
	Title         string              `json:"title,omitempty"`       // Description of the stream (warning: will soon be deprecated in favor of description)
	Description   string              `json:"description,omitempty"` // Description of the stream (previously stream.title)
	FileIndex     uint8               `json:"fileIdx,omitempty"`     // Index of the video file within the torrent (from infoHash); if not specified, the largest file will be selected
	Subtitles     []SubtitleItem      `json:"subtitles,omitempty"`   // Array of Subtitle objects representing subtitles for this stream
	Sources       []string            `json:"sources,omitempty"`     // Array of strings representing torrent tracker URLs and DHT network nodes
	BehaviorHints StreamBehaviorHints `json:"behaviorHints,omitzero"`
}

// StreamBehaviorHints provides additional information about the stream
type StreamBehaviorHints struct {
	CountryWhitelist []string       `json:"countryWhitelist,omitempty"` // Array of ISO 3166-1 alpha-3 country codes in lowercase in which the stream is accessible
	NotWebReady      bool           `json:"notWebReady,omitempty"`      // Set to true if the URL does not support https or is not an MP4 file
	BingeGroup       string         `json:"bingeGroup,omitempty"`       // Identifies the stream's nature within the addon for binge watching
	ProxyHeaders     map[string]any `json:"proxyHeaders,omitempty"`     // Headers that should be used for the stream
	VideoHash        string         `json:"videoHash,omitempty"`        // Calculated OpenSubtitles hash of the video
	VideoSize        int64          `json:"videoSize,omitempty"`        // Size of the video file in bytes
	Filename         string         `json:"filename,omitempty"`         // Filename of the video file
	BingeWatch       bool           `json:"bingeWatch,omitempty"`       // Whether the stream supports binge watching
	AutoPlay         bool           `json:"autoPlay,omitempty"`         // Whether the stream should auto-play
	Proxy            bool           `json:"proxy,omitempty"`            // Whether the stream should be proxied
}

// SubtitleItem represents a subtitle track for a stream
type SubtitleItem struct {
	ID       string `json:"id"`
	URL      string `json:"url"`      // URL to the subtitle file
	Language string `json:"language"` // ISO 639-1 language code
	Label    string `json:"label"`    // Human readable label
}
