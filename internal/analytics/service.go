package analytics

import "github.com/thatsbass/veil/pkg/models"

// Event captures the metadata for a single completed request.
type Event struct {
	UserID   string
	Provider string
	Format   string
	Usage    models.UsageInfo
}

// AnalyticsService records request events asynchronously.
type AnalyticsService interface {
	Track(event Event)
}
