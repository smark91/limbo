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

		info := EvaluateTVRelease(show, []int{2})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		if info.Date == nil || info.Date.Format("2006-01-02") != futureDateStr {
			t.Errorf("expected date %s, got %v", futureDateStr, info.Date)
		}
	})

	t.Run("Fallback to ended series first air date when last is empty", func(t *testing.T) {
		show := &seerr.TVDetail{
			Status:       "Ended",
			FirstAirDate: "2020-01-01",
		}

		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		expectedDate := makeTime("2020-01-01")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})

	t.Run("Per-season air date in future", func(t *testing.T) {
		futureDateStr := time.Now().Add(120 * time.Hour).Format("2006-01-02")
		show := &seerr.TVDetail{
			FirstAirDate: "2020-01-01",
			Seasons: []seerr.TVSeason{
				{
					SeasonNumber: 1,
					AirDate:      futureDateStr,
				},
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

	t.Run("Per-season first episode air date in future", func(t *testing.T) {
		futureDateStr := time.Now().Add(120 * time.Hour).Format("2006-01-02")
		show := &seerr.TVDetail{
			FirstAirDate: "2020-01-01",
			Seasons: []seerr.TVSeason{
				{
					SeasonNumber: 2,
					Episodes: []seerr.TVEpisode{
						{EpisodeNumber: 1, AirDate: futureDateStr},
					},
				},
			},
		}

		info := EvaluateTVRelease(show, []int{2})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		if info.Date == nil || info.Date.Format("2006-01-02") != futureDateStr {
			t.Errorf("expected date %s, got %v", futureDateStr, info.Date)
		}
	})

	t.Run("Ignore future next episode if requested season has already premiered", func(t *testing.T) {
		futureDateStr := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		show := &seerr.TVDetail{
			FirstAirDate: "2020-01-01",
			NextEpisodeToAir: &seerr.TVEpisode{
				AirDate: futureDateStr,
			},
			Seasons: []seerr.TVSeason{
				{
					SeasonNumber: 1,
					AirDate:      "2020-01-01",
				},
			},
		}

		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Air Date" {
			t.Errorf("expected source 'Air Date', got %q", info.Source)
		}
		expectedDate, _ := time.Parse("2006-01-02", "2020-01-01")
		if info.Date == nil || !info.Date.Equal(expectedDate) {
			t.Errorf("expected date %v, got %v", expectedDate, info.Date)
		}
	})

	t.Run("Upcoming status fallback to Unknown when no dates", func(t *testing.T) {
		show := &seerr.TVDetail{
			Status: "Upcoming",
		}
		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Unknown" || info.Date != nil {
			t.Errorf("expected Unknown with nil date, got source=%q, date=%v", info.Source, info.Date)
		}
	})

	t.Run("Default Unknown TV release", func(t *testing.T) {
		show := &seerr.TVDetail{}
		info := EvaluateTVRelease(show, []int{1})
		if info.Source != "Unknown" || info.Date != nil {
			t.Errorf("expected Unknown, got source=%q", info.Source)
		}
	})

	t.Run("Returning Series with unreleased season request", func(t *testing.T) {
		show := &seerr.TVDetail{
			Status:       "Returning Series",
			FirstAirDate: "2020-01-01",
			LastAirDate:  "2021-01-01",
			Seasons: []seerr.TVSeason{
				{
					SeasonNumber: 1,
					AirDate:      "2020-01-01",
				},
			},
		}

		// Requesting Season 2 which does not exist in TMDB yet
		info := EvaluateTVRelease(show, []int{2})
		if info.Source != "Unknown" || info.Date != nil {
			t.Errorf("expected Unknown with nil date for unreleased season, got source=%q, date=%v", info.Source, info.Date)
		}
		if !info.IsUnreleased() {
			t.Error("expected IsUnreleased() to be true for unreleased season 2 request")
		}
	})
}

func TestReleaseInfoIsReleased(t *testing.T) {
	t.Run("Nil Date is not released", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown"}
		if info.IsReleased() {
			t.Error("expected IsReleased() to be false for nil date")
		}
	})

	t.Run("Past Date is released", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Digital"}
		if !info.IsReleased() {
			t.Error("expected IsReleased() to be true for past date")
		}
	})

	t.Run("Future Date is not released", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		info := ReleaseInfo{Date: &future, Source: "Digital"}
		if info.IsReleased() {
			t.Error("expected IsReleased() to be false for future date")
		}
	})
}

func TestReleaseInfoIsSureReleased(t *testing.T) {
	t.Run("Nil Date is not sure released", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown"}
		if info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be false for nil date")
		}
	})

	t.Run("Recent Past Theatrical Date is not sure released", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Theatrical"}
		if info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be false for recent past theatrical date")
		}
	})

	t.Run("Old Past Theatrical Date (>6 months) is sure released", func(t *testing.T) {
		oldPast := time.Now().AddDate(0, -6, -1) // 6 months and 1 day ago
		info := ReleaseInfo{Date: &oldPast, Source: "Theatrical"}
		if !info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be true for theatrical date > 6 months old")
		}
	})

	t.Run("Past Digital Date is sure released", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Digital"}
		if !info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be true for past digital date")
		}
	})

	t.Run("Past Physical Date is sure released", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Physical"}
		if !info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be true for past physical date")
		}
	})

	t.Run("Past Air Date is sure released", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Air Date"}
		if !info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be true for past air date")
		}
	})

	t.Run("Future Digital Date is not sure released", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		info := ReleaseInfo{Date: &future, Source: "Digital"}
		if info.IsSureReleased() {
			t.Error("expected IsSureReleased() to be false for future digital date")
		}
	})
}

func TestReleaseInfoIsUnreleased(t *testing.T) {
	t.Run("Nil date with Released status is NOT unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Released"}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false for status 'Released'")
		}
	})

	t.Run("Nil date with In Production is unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "In Production"}
		if !info.IsUnreleased() {
			t.Error("expected IsUnreleased() true for status 'In Production'")
		}
	})

	t.Run("Nil date with Post Production is unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Post Production"}
		if !info.IsUnreleased() {
			t.Error("expected IsUnreleased() true for status 'Post Production'")
		}
	})

	t.Run("Nil date with Planned is unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Planned"}
		if !info.IsUnreleased() {
			t.Error("expected IsUnreleased() true for status 'Planned'")
		}
	})

	t.Run("Nil date with Upcoming TV is unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Upcoming"}
		if !info.IsUnreleased() {
			t.Error("expected IsUnreleased() true for status 'Upcoming'")
		}
	})

	t.Run("Nil date with Returning Series is NOT unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Returning Series"}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false for status 'Returning Series'")
		}
	})

	t.Run("Nil date with Ended is NOT unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Ended"}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false for status 'Ended'")
		}
	})

	t.Run("Nil date with Canceled is NOT unreleased", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: "Canceled"}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false for status 'Canceled'")
		}
	})

	t.Run("Nil date with empty status is NOT unreleased (safe default)", func(t *testing.T) {
		info := ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: ""}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false for empty status")
		}
	})

	t.Run("Non-nil date always returns false regardless of status", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		info := ReleaseInfo{Date: &past, Source: "Unknown", MediaStatus: "In Production"}
		if info.IsUnreleased() {
			t.Error("expected IsUnreleased() false when Date is not nil")
		}
	})
}



func TestParseReleaseDateEdgeCases(t *testing.T) {
	t.Run("Fallback parsing formats", func(t *testing.T) {
		d1 := parseReleaseDate("2026-04-15T14:59:24.000Z")
		if d1 == nil || d1.Year() != 2026 {
			t.Errorf("expected parsed date, got %v", d1)
		}

		d2 := parseReleaseDate("2026-04-15T14:59:24Z")
		if d2 == nil || d2.Year() != 2026 {
			t.Errorf("expected parsed date, got %v", d2)
		}

		d3 := parseReleaseDate("2026-04-15")
		if d3 == nil || d3.Year() != 2026 {
			t.Errorf("expected parsed date, got %v", d3)
		}

		d4 := parseReleaseDate("invalid-date")
		if d4 != nil {
			t.Errorf("expected nil for invalid date string, got %v", d4)
		}

		d5 := parseSimpleDate("invalid-date")
		if d5 != nil {
			t.Errorf("expected nil for invalid simple date string, got %v", d5)
		}
	})
}
