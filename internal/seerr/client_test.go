package seerr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetTVDetail(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tv/123" {
			t.Errorf("expected path /api/v1/tv/123, got %q", r.URL.Path)
		}

		detail := TVDetail{
			ID:         123,
			Name:       "Test Show",
			PosterPath: "/tv_poster.jpg",
			Status:     "Returning Series",
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
	detail, err := client.GetTVDetail(ctx, 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.Name != "Test Show" {
		t.Errorf("expected Name 'Test Show', got %q", detail.Name)
	}
	if detail.PosterPath != "/tv_poster.jpg" {
		t.Errorf("expected poster path '/tv_poster.jpg', got %q", detail.PosterPath)
	}
}

func TestPing(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/status" {
			t.Errorf("expected path /api/v1/status, got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteRequest(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %q", r.Method)
		}
		if r.URL.Path != "/api/v1/request/999" {
			t.Errorf("expected path /api/v1/request/999, got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	err := client.DeleteRequest(ctx, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteMedia(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected method DELETE, got %q", r.Method)
		}
		if r.URL.Path != "/api/v1/media/888" {
			t.Errorf("expected path /api/v1/media/888, got %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	err := client.DeleteMedia(ctx, 888)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRequest(t *testing.T) {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/request/111" {
			t.Errorf("expected path /api/v1/request/111, got %q", r.URL.Path)
		}

		req := SeerrRequest{
			ID:        111,
			Status:    2,
			MediaType: "movie",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: apiKey,
	}

	client := NewClient(cfg)
	ctx := context.Background()
	req, err := client.GetRequest(ctx, 111)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.ID != 111 {
		t.Errorf("expected ID 111, got %d", req.ID)
	}
}

func TestGetApprovedRequestsPagination(t *testing.T) {
	apiKey := "test-api-key"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")

		var resp RequestsResponse
		if calls == 1 {
			resp = RequestsResponse{
				PageInfo: PageInfo{Pages: 2, Page: 1, Results: 2},
				Results: []SeerrRequest{
					{ID: 1, MediaType: "movie"},
				},
			}
		} else {
			resp = RequestsResponse{
				PageInfo: PageInfo{Pages: 2, Page: 2, Results: 2},
				Results: []SeerrRequest{
					{ID: 2, MediaType: "tv"},
				},
			}
		}
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
	if reqs[0].ID != 1 || reqs[1].ID != 2 {
		t.Errorf("incorrect requests returned: %+v", reqs)
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls, got %d", calls)
	}
}

func TestClientErrors(t *testing.T) {
	t.Run("PublicURL helper", func(t *testing.T) {
		client := NewClient(&config.Config{SeerrPublicURL: "http://public"})
		if client.PublicURL() != "http://public" {
			t.Errorf("expected public url 'http://public', got %q", client.PublicURL())
		}
	})

	t.Run("doGet HTTP Error", func(t *testing.T) {
		client := NewClient(&config.Config{SeerrURL: "http://127.0.0.1:9999"})
		ctx := context.Background()
		_, err := client.GetMovieDetail(ctx, 123)
		if err == nil {
			t.Errorf("expected error for invalid URL, got nil")
		}
	})

	t.Run("doGet Non-200 Status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found Error Message"))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetMovieDetail(ctx, 123)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "seerr API returned 404") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("doGet JSON Parse Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid-json"))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetMovieDetail(ctx, 123)
		if err == nil {
			t.Errorf("expected JSON parse error, got nil")
		}
	})

	t.Run("doGet HTTP Request Context Cancelled", func(t *testing.T) {
		client := NewClient(&config.Config{SeerrURL: "http://127.0.0.1:1234"})
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		_, err := client.GetMovieDetail(ctx, 123)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("doDelete HTTP Error", func(t *testing.T) {
		client := NewClient(&config.Config{SeerrURL: "http://127.0.0.1:9999"})
		ctx := context.Background()
		err := client.DeleteRequest(ctx, 123)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("doDelete Non-200 Status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server Error"))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		err := client.DeleteRequest(ctx, 123)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "seerr API returned 500") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("GetApprovedRequests error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetApprovedRequests(ctx)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("GetApprovedRequests parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid-json`))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetApprovedRequests(ctx)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("GetTVDetail HTTP Error", func(t *testing.T) {
		client := NewClient(&config.Config{SeerrURL: "http://127.0.0.1:9999"})
		ctx := context.Background()
		_, err := client.GetTVDetail(ctx, 123)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("GetTVDetail parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid-json`))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetTVDetail(ctx, 123)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("GetRequest parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid-json`))
		}))
		defer server.Close()

		client := NewClient(&config.Config{SeerrURL: server.URL})
		ctx := context.Background()
		_, err := client.GetRequest(ctx, 123)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}
