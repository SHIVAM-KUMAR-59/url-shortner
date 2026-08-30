package analytics

import (
	"context"
	"time"
)

type Reader interface {
	GetStats(ctx context.Context, shortCode string) (totalClicks int64, lastClicked *time.Time, err error)
}
