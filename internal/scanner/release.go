package scanner

import (
	"strings"
	"time"

	"limbo/internal/config"
	"limbo/internal/seerr"
)

// ReleaseInfo holds the evaluated release date and its source.
type ReleaseInfo struct {
	Date   *time.Time
	Source string // "Digital", "Physical", "Theatrical", "Air Date", "Unknown"
}

// EvaluateMovieRelease determines the best release date for a movie.
// Priority: Digital (type 4) > Physical (type 5) > Theatrical (type 3) fallback.
// Only considers releases matching the configured country.
func EvaluateMovieRelease(movie *seerr.MovieDetail, cfg *config.Config) ReleaseInfo {
	preferredCountry := strings.ToUpper(cfg.ReleaseCountry)

	var (
		digitalLocal, digitalGlobal       *time.Time
		physicalLocal, physicalGlobal     *time.Time
		theatricalLocal, theatricalGlobal *time.Time
	)

	for _, countryReleases := range movie.ReleaseDates.Results {
		isPreferred := strings.ToUpper(countryReleases.ISO31661) == preferredCountry

		for _, rd := range countryReleases.ReleaseDates {
			parsed := parseReleaseDate(rd.ReleaseDate)
			if parsed == nil {
				continue
			}

			switch rd.Type {
			case 4: // Digital
				if isPreferred && (digitalLocal == nil || parsed.Before(*digitalLocal)) {
					digitalLocal = parsed
				}
				if digitalGlobal == nil || parsed.Before(*digitalGlobal) {
					digitalGlobal = parsed
				}
			case 5: // Physical
				if isPreferred && (physicalLocal == nil || parsed.Before(*physicalLocal)) {
					physicalLocal = parsed
				}
				if physicalGlobal == nil || parsed.Before(*physicalGlobal) {
					physicalGlobal = parsed
				}
			case 3: // Theatrical
				if isPreferred && (theatricalLocal == nil || parsed.Before(*theatricalLocal)) {
					theatricalLocal = parsed
				}
				if theatricalGlobal == nil || parsed.Before(*theatricalGlobal) {
					theatricalGlobal = parsed
				}
			}
		}
	}

	// Priority Decision
	if digitalLocal != nil {
		return ReleaseInfo{Date: digitalLocal, Source: "Digital"}
	}
	if digitalGlobal != nil {
		return ReleaseInfo{Date: digitalGlobal, Source: "Digital"}
	}
	if physicalLocal != nil {
		return ReleaseInfo{Date: physicalLocal, Source: "Physical"}
	}
	if physicalGlobal != nil {
		return ReleaseInfo{Date: physicalGlobal, Source: "Physical"}
	}
	if theatricalLocal != nil {
		return ReleaseInfo{Date: theatricalLocal, Source: "Theatrical"}
	}
	if theatricalGlobal != nil {
		return ReleaseInfo{Date: theatricalGlobal, Source: "Theatrical"}
	}

	// Fallback to generic release date if all else fails
	if movie.ReleaseDate != "" {
		if t := parseSimpleDate(movie.ReleaseDate); t != nil {
			return ReleaseInfo{Date: t, Source: "Theatrical"}
		}
	}

	return ReleaseInfo{Date: nil, Source: "Unknown"}
}

// EvaluateTVRelease determines the next relevant air date for a TV show.
// Checks show status, firstAirDate, and per-season episode air dates.
func EvaluateTVRelease(show *seerr.TVDetail, requestedSeasons []int) ReleaseInfo {
	now := time.Now()

	// 1. If FirstAirDate is in the future, it's the most reliable premiere date
	if show.FirstAirDate != "" {
		if t := parseSimpleDate(show.FirstAirDate); t != nil && t.After(now) {
			return ReleaseInfo{Date: t, Source: "Air Date"}
		}
	}

	// 2. Check for the next specific episode to air (covers returning series)
	if show.NextEpisodeToAir != nil && show.NextEpisodeToAir.AirDate != "" {
		if t := parseReleaseDate(show.NextEpisodeToAir.AirDate); t != nil && t.After(now) {
			return ReleaseInfo{Date: t, Source: "Air Date"}
		}
	}

	// 3. Check per-season air dates for requested seasons (fallbacks)
	var earliestFuture *time.Time
	for _, reqSeason := range requestedSeasons {
		for _, season := range show.Seasons {
			if season.SeasonNumber != reqSeason {
				continue
			}

			// Check season air date
			if season.AirDate != "" {
				if t := parseSimpleDate(season.AirDate); t != nil && t.After(now) {
					if earliestFuture == nil || t.Before(*earliestFuture) {
						earliestFuture = t
					}
				}
			}

			// Check first episode air date for more precision
			if len(season.Episodes) > 0 && season.Episodes[0].AirDate != "" {
				if t := parseSimpleDate(season.Episodes[0].AirDate); t != nil && t.After(now) {
					if earliestFuture == nil || t.Before(*earliestFuture) {
						earliestFuture = t
					}
				}
			}
		}
	}

	if earliestFuture != nil {
		return ReleaseInfo{Date: earliestFuture, Source: "Air Date"}
	}

	// 4. Fallback for Ended/Canceled shows or previously released content
	fallbackDate := show.LastAirDate
	if fallbackDate == "" {
		fallbackDate = show.FirstAirDate
	}

	if fallbackDate != "" {
		if t := parseSimpleDate(fallbackDate); t != nil {
			return ReleaseInfo{Date: t, Source: "Air Date"}
		}
	}

	// 5. If explicitly "Upcoming" but no dates found, it's truly Unknown
	if show.Status == "Upcoming" {
		return ReleaseInfo{Date: nil, Source: "Unknown"}
	}

	return ReleaseInfo{Date: nil, Source: "Unknown"}
}

// IsReleased returns true if the release date is in the past.
func (r ReleaseInfo) IsReleased() bool {
	if r.Date == nil {
		return false
	}
	return r.Date.Before(time.Now())
}

// parseReleaseDate handles ISO 8601 dates like "2025-03-15T00:00:00.000Z"
func parseReleaseDate(s string) *time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}

// parseSimpleDate handles "2025-03-15" format dates.
func parseSimpleDate(s string) *time.Time {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}
