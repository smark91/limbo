package seerr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"limbo/internal/config"
)

func TestGetApprovedRequests(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-Api-Key") != apiKey {
			t.Errorf("expected API Key header %q, got %q", apiKey, r.Header.Get("X-Api-Key"))
		}
		if r.URL.Path != "/api/v1/request" {
			t.Errorf("expected request path /api/v1/request, got %q", r.URL.Path)
		}

		// Mock response
		resp := RequestsResponse{
			PageInfo: PageInfo{Pages: 1, Page: 1, Results: 1},
			Results: []SeerrRequest{
				{
					ID:        42,
					Status:    2,
					MediaType: "movie",
					Media: Media{
						ID:     100,
						TmdbID: 550,
					},
					CreatedAt: "2026-05-20T12:00:00Z",
					UpdatedAt: "2026-05-20T12:05:00Z",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	reqs, err := client.GetApprovedRequests(ctx)
	if err != nil {
		t.Fatalf("unexpected error fetching approved requests: %v", err)
	}

	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].ID != 42 {
		t.Errorf("expected request ID 42, got %d", reqs[0].ID)
	}
	if reqs[0].Media.TmdbID != 550 {
		t.Errorf("expected TMDB ID 550, got %d", reqs[0].Media.TmdbID)
	}
}

func TestGetMovieDetail(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/movie/550" {
			t.Errorf("expected path /api/v1/movie/550, got %q", r.URL.Path)
		}

		detail := MovieDetail{
			ID:          550,
			Title:       "Fight Club",
			PosterPath:  "/poster.jpg",
			Status:      "Released",
			ReleaseDate: "1999-10-15",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	detail, err := client.GetMovieDetail(ctx, 550)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.Title != "Fight Club" {
		t.Errorf("expected title 'Fight Club', got %q", detail.Title)
	}
	if detail.PosterPath != "/poster.jpg" {
		t.Errorf("expected poster path '/poster.jpg', got %q", detail.PosterPath)
	}
}
