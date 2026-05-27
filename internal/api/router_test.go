package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"limbo/internal/config"
	"limbo/internal/scanner"
	"limbo/internal/seerr"
)

func TestRouterAndSPA(t *testing.T) {
	db := setupTestDB(t)

	// Mock Seerr API Server so scanner behaves
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		SeerrURL:    server.URL,
		SeerrAPIKey: "test-key",
	}

	sc := scanner.New(cfg, db, seerr.NewClient(cfg))

	// Mock filesystem for embedded frontend files
	mockFS := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("SPA index content")},
		"manifest.json":  &fstest.MapFile{Data: []byte("manifest content")},
		"assets/logo.svg": &fstest.MapFile{Data: []byte("svg logo")},
	}

	router := NewRouter(Deps{
		Config:  cfg,
		DB:      db,
		Scanner: sc,
		Seerr:   seerr.NewClient(cfg),
	}, http.FS(mockFS))

	// Test 1: Hit API route
	t.Run("API Health Route", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	// Test 2: Hit exact static file
	t.Run("Serve Exact File", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/manifest.json", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body, _ := io.ReadAll(rr.Result().Body)
		if string(body) != "manifest content" {
			t.Errorf("expected 'manifest content', got %q", string(body))
		}
	})

	// Test 3: Hit nonexistent file fallback to SPA routing (index.html)
	t.Run("Serve Nonexistent Path Fallback", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/some/random/route", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body, _ := io.ReadAll(rr.Result().Body)
		if string(body) != "SPA index content" {
			t.Errorf("expected fallback 'SPA index content', got %q", string(body))
		}
	})

	// Test 4: OPTIONS CORS preflight check
	t.Run("CORS Preflight", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/health", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for OPTIONS, got %d", rr.Code)
		}

		origin := rr.Header().Get("Access-Control-Allow-Origin")
		if origin != "*" {
			t.Errorf("expected Access-Control-Allow-Origin to be '*', got %q", origin)
		}
	})
}
