package database

import "time"

// Triage status constants
const (
	StatusPending        = "PENDING"
	StatusWaitingRelease = "WAITING_RELEASE"
	StatusNotAvailable   = "NOT_AVAILABLE"
	StatusCompleted      = "COMPLETED"
)

// AllStatuses returns all valid triage statuses for validation.
func AllStatuses() []string {
	return []string{
		StatusPending,
		StatusWaitingRelease,
		StatusNotAvailable,
		StatusCompleted,
	}
}

// TriageEntry tracks the triage state of an unfulfilled Seerr request.
type TriageEntry struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	SeerrRequestID int        `gorm:"uniqueIndex" json:"seerrRequestId"`
	MediaID        int        `json:"mediaId"`
	TmdbID         int        `json:"tmdbId"`
	PosterPath     string     `json:"posterPath,omitempty"`
	MediaType      string     `json:"mediaType"`      // "movie" or "tv"
	Title          string     `json:"title"`           // Cached title from Seerr
	Status         string     `gorm:"default:PENDING" json:"status"`
	Notes          *string    `json:"notes,omitempty"`
	Reason         *string    `json:"reason,omitempty"`        // Why not-available
	ReleaseDate    *time.Time `json:"releaseDate,omitempty"`
	ReleaseSource  *string    `json:"releaseSource,omitempty"` // "Digital", "Physical", "Theatrical", "Unknown"
	ServiceURL     string     `json:"serviceUrl,omitempty"`
	NotifiedAt     *time.Time `json:"notifiedAt,omitempty"`
	FulfilledAt    *time.Time `json:"fulfilledAt,omitempty"`
	SeerrCreatedAt time.Time  `gorm:"column:seerr_created_at" json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
// SystemMetadata stores application-wide metadata (e.g., last scan time).
type SystemMetadata struct {
	Key       string    `gorm:"primaryKey"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
