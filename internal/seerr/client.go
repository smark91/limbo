package seerr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smark91/limbo/internal/config"
)

// Client communicates with the Seerr API.
type Client struct {
	baseURL    string
	publicURL  string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Seerr API client.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:   cfg.SeerrURL,
		publicURL: cfg.SeerrPublicURL,
		apiKey:    cfg.SeerrAPIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PublicURL returns the public-facing Seerr URL for external links.
func (c *Client) PublicURL() string {
	return c.publicURL
}

// --- API Response Types ---

// RequestsResponse is the paginated list from /api/v1/request.
type RequestsResponse struct {
	PageInfo PageInfo       `json:"pageInfo"`
	Results  []SeerrRequest `json:"results"`
}

type PageInfo struct {
	Pages   int `json:"pages"`
	Page    int `json:"page"`
	Results int `json:"results"`
}

// SeerrRequest represents a single request from Seerr.
type SeerrRequest struct {
	ID          int    `json:"id"`
	Status      int    `json:"status"`    // 1=pending, 2=approved, 3=declined
	Is4K        bool   `json:"is4k"`
	MediaType   string `json:"type"`      // "movie" or "tv"
	Media       Media  `json:"media"`
	Seasons     []struct {
		SeasonNumber int `json:"seasonNumber"`
	} `json:"seasons"`
	RequestedBy RequestedBy `json:"requestedBy"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

type Media struct {
	ID              int           `json:"id"`
	TmdbID          int           `json:"tmdbId"`
	TvdbID          *int          `json:"tvdbId"`
	Status          int           `json:"status"` // 1=unknown, 2=pending, 3=processing, 4=partially_available, 5=available
	Status4k        int           `json:"status4k"`
	MediaType       string        `json:"mediaType"`
	DownloadStatus  []interface{} `json:"downloadStatus"`
	RatingKey4k     *string       `json:"ratingKey4k"`
	JellyfinMediaId   *string     `json:"jellyfinMediaId"`
	JellyfinMediaId4k *string     `json:"jellyfinMediaId4k"`
	ServiceURL      string        `json:"serviceUrl"`
	Seasons         []MediaSeason `json:"seasons"`
}

type MediaSeason struct {
	ID           int `json:"id"`
	SeasonNumber int `json:"seasonNumber"`
	Status       int `json:"status"` // 1=unknown, 2=pending, 3=processing, 4=partially_available, 5=available
	Status4k     int `json:"status4k"`
}

type RequestedBy struct {
	ID          int    `json:"id"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}

// MovieDetail from /api/v1/movie/{tmdbId}
type MovieDetail struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	PosterPath  string `json:"posterPath"`
	Status      string `json:"status"`      // "Released", "In Production", etc.
	ReleaseDate string `json:"releaseDate"` // "2025-03-15"
	ReleaseDates struct {
		Results []ReleaseDateCountry `json:"releases"`
	} `json:"releases"`
}

type ReleaseDateCountry struct {
	ISO31661     string        `json:"iso_3166_1"`
	ReleaseDates []ReleaseDate `json:"release_dates"`
}

type ReleaseDate struct {
	Type        int    `json:"type"` // 1=Premiere, 2=Theatrical (limited), 3=Theatrical, 4=Digital, 5=Physical, 6=TV
	ReleaseDate string `json:"release_date"`
}

// TVDetail from /api/v1/tv/{tmdbId}
type TVDetail struct {
	ID               int           `json:"id"`
	Name             string        `json:"name"`
	PosterPath       string        `json:"posterPath"`
	Status           string        `json:"status"` // "Returning Series", "Ended", "Upcoming", etc.
	FirstAirDate     string        `json:"firstAirDate"`
	LastAirDate      string        `json:"lastAirDate"`
	NextEpisodeToAir *TVEpisode    `json:"nextEpisodeToAir"`
	LastEpisodeToAir *TVEpisode    `json:"lastEpisodeToAir"`
	Seasons          []TVSeason    `json:"seasons"`
}

type TVSeason struct {
	SeasonNumber int    `json:"seasonNumber"`
	AirDate      string `json:"airDate"`
	EpisodeCount int    `json:"episodeCount"`
	Episodes     []TVEpisode `json:"episodes,omitempty"`
}

type TVEpisode struct {
	EpisodeNumber int    `json:"episodeNumber"`
	SeasonNumber  int    `json:"seasonNumber"`
	AirDate       string `json:"airDate"`
}

// --- API Methods ---

// GetApprovedRequests fetches all approved requests from Seerr.
func (c *Client) GetApprovedRequests(ctx context.Context) ([]SeerrRequest, error) {
	var all []SeerrRequest
	page := 1

	for {
		url := fmt.Sprintf("%s/api/v1/request?filter=approved&take=100&skip=%d&sort=added", c.baseURL, (page-1)*100)
		resp, err := c.doGet(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("fetching requests page %d: %w", page, err)
		}

		var result RequestsResponse
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("parsing requests page %d: %w", page, err)
		}

		all = append(all, result.Results...)

		if page >= result.PageInfo.Pages || len(result.Results) == 0 {
			break
		}
		page++
	}

	return all, nil
}

// GetMovieDetail fetches movie detail from Seerr by TMDB ID.
func (c *Client) GetMovieDetail(ctx context.Context, tmdbID int) (*MovieDetail, error) {
	url := fmt.Sprintf("%s/api/v1/movie/%d", c.baseURL, tmdbID)
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var detail MovieDetail
	if err := json.Unmarshal(resp, &detail); err != nil {
		return nil, fmt.Errorf("parsing movie detail: %w", err)
	}
	return &detail, nil
}

// GetTVDetail fetches TV show detail from Seerr by TMDB ID.
func (c *Client) GetTVDetail(ctx context.Context, tmdbID int) (*TVDetail, error) {
	url := fmt.Sprintf("%s/api/v1/tv/%d", c.baseURL, tmdbID)
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var detail TVDetail
	if err := json.Unmarshal(resp, &detail); err != nil {
		return nil, fmt.Errorf("parsing TV detail: %w", err)
	}
	return &detail, nil
}

// doGet performs an authenticated GET request.
func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("403 Forbidden: check if SEERR_API_KEY is correct and has administrator permissions")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("401 Unauthorized: check if SEERR_API_KEY is correct")
		}
		limit := 200
		if len(body) < limit {
			limit = len(body)
		}
		return nil, fmt.Errorf("seerr API returned %d: %s", resp.StatusCode, string(body[:limit]))
	}

	return body, nil
}

// Ping checks if Seerr is reachable and responding correctly.
func (c *Client) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/status", c.baseURL)
	_, err := c.doGet(ctx, url)
	return err
}

// DeleteRequest deletes a request from Seerr by its request ID.
func (c *Client) DeleteRequest(ctx context.Context, requestID int) error {
	url := fmt.Sprintf("%s/api/v1/request/%d", c.baseURL, requestID)
	return c.doDelete(ctx, url)
}

// DeleteMedia clears a media metadata record in Seerr by its media database ID.
func (c *Client) DeleteMedia(ctx context.Context, mediaID int) error {
	url := fmt.Sprintf("%s/api/v1/media/%d", c.baseURL, mediaID)
	return c.doDelete(ctx, url)
}

// doDelete performs an authenticated DELETE request.
func (c *Client) doDelete(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("403 Forbidden: check if SEERR_API_KEY is correct and has administrator permissions")
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("401 Unauthorized: check if SEERR_API_KEY is correct")
		}
		body, _ := io.ReadAll(resp.Body)
		limit := 200
		if len(body) < limit {
			limit = len(body)
		}
		return fmt.Errorf("seerr API returned %d: %s", resp.StatusCode, string(body[:limit]))
	}

	return nil
}

// GetRequest fetches details of a single request from Seerr by its request ID.
func (c *Client) GetRequest(ctx context.Context, requestID int) (*SeerrRequest, error) {
	url := fmt.Sprintf("%s/api/v1/request/%d", c.baseURL, requestID)
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return nil, err
	}
	var req SeerrRequest
	if err := json.Unmarshal(resp, &req); err != nil {
		return nil, fmt.Errorf("parsing request detail: %w", err)
	}
	return &req, nil
}

// SeerrStatus represents the response from /api/v1/status.
type SeerrStatus struct {
	Version string `json:"version"`
}

// GetVersion fetches the Seerr application version.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v1/status", c.baseURL)
	resp, err := c.doGet(ctx, url)
	if err != nil {
		return "", err
	}
	var status SeerrStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return "", err
	}
	return status.Version, nil
}
