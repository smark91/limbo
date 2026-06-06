package scanner

import (
	"strings"
	"time"

	"github.com/smark91/limbo/internal/config"
	"github.com/smark91/limbo/internal/seerr"
)

// ReleaseInfo holds the evaluated release date and its source.
type ReleaseInfo struct {
	Date        *time.Time
	Source      string // "Digital", "Physical", "Theatrical", "Air Date", "Unknown"
	MediaStatus string // Raw status string from TMDB/Seerr (e.g. "Released", "In Production", "Upcoming")
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
		return ReleaseInfo{Date: digitalLocal, Source: "Digital", MediaStatus: movie.Status}
	}
	if digitalGlobal != nil {
		return ReleaseInfo{Date: digitalGlobal, Source: "Digital", MediaStatus: movie.Status}
	}
	if physicalLocal != nil {
		return ReleaseInfo{Date: physicalLocal, Source: "Physical", MediaStatus: movie.Status}
	}
	if physicalGlobal != nil {
		return ReleaseInfo{Date: physicalGlobal, Source: "Physical", MediaStatus: movie.Status}
	}
	if theatricalLocal != nil {
		return ReleaseInfo{Date: theatricalLocal, Source: "Theatrical", MediaStatus: movie.Status}
	}
	if theatricalGlobal != nil {
		return ReleaseInfo{Date: theatricalGlobal, Source: "Theatrical", MediaStatus: movie.Status}
	}

	// Fallback to generic release date if all else fails
	if movie.ReleaseDate != "" {
		if t := parseSimpleDate(movie.ReleaseDate); t != nil {
			return ReleaseInfo{Date: t, Source: "Theatrical", MediaStatus: movie.Status}
		}
	}

	return ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: movie.Status}
}

// EvaluateTVRelease determines the next relevant air date for a TV show.
// Checks show status, firstAirDate, and per-season episode air dates.
func EvaluateTVRelease(show *seerr.TVDetail, requestedSeasons []int) ReleaseInfo {
	now := time.Now()

	// 1. If FirstAirDate is in the future, it's the most reliable premiere date
	if show.FirstAirDate != "" {
		if t := parseSimpleDate(show.FirstAirDate); t != nil && t.After(now) {
			return ReleaseInfo{Date: t, Source: "Air Date", MediaStatus: show.Status}
		}
	}

	// Determine if any of the requested seasons have already premiered.
	// If a requested season has premiered, we should ignore any future NextEpisodeToAir.
	requestedSeasonPremiered := false
	if show.Status == "Ended" || show.Status == "Canceled" {
		requestedSeasonPremiered = true
	}
	var earliestFutureSeasonDate *time.Time

	for _, reqSeason := range requestedSeasons {
		seasonFound := false
		seasonHasDate := false
		for _, season := range show.Seasons {
			if season.SeasonNumber != reqSeason {
				continue
			}
			seasonFound = true

			// Check if season has a known air date
			if season.AirDate != "" {
				if t := parseSimpleDate(season.AirDate); t != nil {
					seasonHasDate = true
					if t.After(now) {
						if earliestFutureSeasonDate == nil || t.Before(*earliestFutureSeasonDate) {
							earliestFutureSeasonDate = t
						}
					} else {
						requestedSeasonPremiered = true
					}
				}
			} else if len(season.Episodes) > 0 && season.Episodes[0].AirDate != "" {
				if t := parseSimpleDate(season.Episodes[0].AirDate); t != nil {
					seasonHasDate = true
					if t.After(now) {
						if earliestFutureSeasonDate == nil || t.Before(*earliestFutureSeasonDate) {
							earliestFutureSeasonDate = t
						}
					} else {
						requestedSeasonPremiered = true
					}
				}
			}
		}

		// Fallback if season is not in show.Seasons or has no date
		if (!seasonFound || !seasonHasDate) && show.FirstAirDate != "" {
			if t := parseSimpleDate(show.FirstAirDate); t != nil && !t.After(now) {
				if reqSeason == 1 {
					requestedSeasonPremiered = true
				}
			}
		}
	}

	// 2. Check for the next specific episode to air.
	// But ONLY if we haven't already confirmed that a requested season has premiered!
	if !requestedSeasonPremiered && show.NextEpisodeToAir != nil && show.NextEpisodeToAir.AirDate != "" {
		if t := parseReleaseDate(show.NextEpisodeToAir.AirDate); t != nil && t.After(now) {
			return ReleaseInfo{Date: t, Source: "Air Date", MediaStatus: show.Status}
		}
	}

	// 3. Return the earliest future season date if we found one
	if earliestFutureSeasonDate != nil {
		return ReleaseInfo{Date: earliestFutureSeasonDate, Source: "Air Date", MediaStatus: show.Status}
	}

	// 4. Fallback for Ended/Canceled shows or previously released content.
	// But ONLY if we didn't request specific seasons that haven't premiered yet!
	if len(requestedSeasons) == 0 || requestedSeasonPremiered {
		fallbackDate := show.LastAirDate
		if fallbackDate == "" {
			fallbackDate = show.FirstAirDate
		}

		if fallbackDate != "" {
			if t := parseSimpleDate(fallbackDate); t != nil {
				return ReleaseInfo{Date: t, Source: "Air Date", MediaStatus: show.Status}
			}
		}
	}

	// 5. No dates found at all — return Unknown with status for downstream decision-making.
	// If requested seasons have not premiered, override MediaStatus to "Upcoming" so downstream
	// logic (IsUnreleased) treats it as WAITING_RELEASE.
	mediaStatus := show.Status
	if len(requestedSeasons) > 0 && !requestedSeasonPremiered {
		mediaStatus = "Upcoming"
	}
	return ReleaseInfo{Date: nil, Source: "Unknown", MediaStatus: mediaStatus}
}

// IsReleased returns true if the release date is in the past.
func (r ReleaseInfo) IsReleased() bool {
	if r.Date == nil {
		return false
	}
	return r.Date.Before(time.Now())
}

// IsSureReleased returns true if the media has had a sure release (Digital, Physical, or Air Date) in the past.
func (r ReleaseInfo) IsSureReleased() bool {
	if r.Date == nil {
		return false
	}
	if r.Source == "Theatrical" {
		sixMonthsAgo := time.Now().AddDate(0, -6, 0)
		return r.Date.Before(sixMonthsAgo)
	}
	return r.Date.Before(time.Now())
}

// IsUnreleased returns true when there is no known release date but the media
// status signals it hasn't been released yet. This is used as a fallback to
// decide WAITING_RELEASE vs PENDING when Date is nil.
//
// Movies: anything other than "Released" is considered unreleased.
// TV: only "Upcoming" is considered unreleased (other statuses like
// "Returning Series" / "Ended" / "Canceled" are treated as already aired).
func (r ReleaseInfo) IsUnreleased() bool {
	if r.Date != nil {
		// Date-based logic takes precedence; this helper is for nil-date cases only.
		return false
	}
	switch r.MediaStatus {
	case "Released":
		// Movie is definitely out.
		return false
	case "Returning Series", "Ended", "Canceled", "Cancellation Requested":
		// TV show has aired; treat as released.
		return false
	case "":
		// No status information — assume released to avoid keeping in WAITING_RELEASE forever.
		return false
	default:
		// "In Production", "Post Production", "Planned", "Upcoming", etc.
		return true
	}
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
