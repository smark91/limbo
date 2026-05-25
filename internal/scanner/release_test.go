package scanner

import (
	"testing"
	"time"

	"limbo/internal/config"
	"limbo/internal/seerr"
)

func TestEvaluateMovieRelease(t *testing.T) {
	cfg := &config.Config{ReleaseCountry: "US"}

	// Helper to create times
	makeTime := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}

	t.Run("Digital release priority US", func(t *testing.T) {
		movie := &seerr.MovieDetail{
			Title: "Test Movie",
			ReleaseDates: struct {
				Results []seerr.ReleaseDateCountry `json:"releases"`
			}{
				Results: []seerr.ReleaseDateCountry{
					{
						ISO31661: "US",
						ReleaseDates: []seerr.ReleaseDate{
							{Type: 3, ReleaseDate: "2025-01-01T00:00:00.000Z"}, // Theatrical
							{Type: 4, ReleaseDate: "2025-02-01T00:00:00.000Z"}, // Digital
						},
					},
				},
			},
		}

		info := EvaluateMovieRelease(movie, cfg)
		if info.Source != "Digital" {
			t.Errorf("expected source 'Digital', got %q", info.Source)
		}
		expectedDate := makeTime("2025-02-01")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})

	t.Run("Physical fallback when no digital", func(t *testing.T) {
		movie := &seerr.MovieDetail{
			Title: "Test Movie 2",
			ReleaseDates: struct {
				Results []seerr.ReleaseDateCountry `json:"releases"`
			}{
				Results: []seerr.ReleaseDateCountry{
					{
						ISO31661: "US",
						ReleaseDates: []seerr.ReleaseDate{
							{Type: 3, ReleaseDate: "2025-01-01T00:00:00.000Z"}, // Theatrical
							{Type: 5, ReleaseDate: "2025-03-01T00:00:00.000Z"}, // Physical
						},
					},
				},
			},
		}

		info := EvaluateMovieRelease(movie, cfg)
		if info.Source != "Physical" {
			t.Errorf("expected source 'Physical', got %q", info.Source)
		}
		expectedDate := makeTime("2025-03-01")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})

	t.Run("Global fallback when no local release", func(t *testing.T) {
		movie := &seerr.MovieDetail{
			Title: "Test Movie 3",
			ReleaseDates: struct {
				Results []seerr.ReleaseDateCountry `json:"releases"`
			}{
				Results: []seerr.ReleaseDateCountry{
					{
						ISO31661: "UK",
						ReleaseDates: []seerr.ReleaseDate{
							{Type: 4, ReleaseDate: "2025-04-15T00:00:00.000Z"}, // Digital UK
						},
					},
				},
			},
		}

		info := EvaluateMovieRelease(movie, cfg)
		if info.Source != "Digital" {
			t.Errorf("expected source 'Digital', got %q", info.Source)
		}
		expectedDate := makeTime("2025-04-15")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})

	t.Run("Fallback to simple release date", func(t *testing.T) {
		movie := &seerr.MovieDetail{
			Title:       "Test Movie 4",
			ReleaseDate: "2025-05-20",
		}

		info := EvaluateMovieRelease(movie, cfg)
		if info.Source != "Theatrical" {
			t.Errorf("expected source 'Theatrical', got %q", info.Source)
		}
		expectedDate := makeTime("2025-05-20")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})
}

func TestEvaluateTVRelease(t *testing.T) {
	// Helper to create times
	makeTime := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}

	t.Run("Future first air date", func(t *testing.T) {
		futureDateStr := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		show := &seerr.TVDetail{
			FirstAirDate: futureDateStr,
		}

		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		if info.Date == nil || info.Date.Format("2006-01-02") != futureDateStr {
			t.Errorf("expected date %s, got %v", futureDateStr, info.Date)
		}
	})

	t.Run("Next episode to air", func(t *testing.T) {
		futureDateStr := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		show := &seerr.TVDetail{
			FirstAirDate: "2020-01-01",
			NextEpisodeToAir: &seerr.TVEpisode{
				AirDate: futureDateStr,
			},
		}

		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		if info.Date == nil || info.Date.Format("2006-01-02") != futureDateStr {
			t.Errorf("expected date %s, got %v", futureDateStr, info.Date)
		}
	})

	t.Run("Fallback to ended series last air date", func(t *testing.T) {
		show := &seerr.TVDetail{
			Status:       "Ended",
			FirstAirDate: "2020-01-01",
			LastAirDate:  "2020-05-01",
		}

		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		expectedDate := makeTime("2020-05-01")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})
}
