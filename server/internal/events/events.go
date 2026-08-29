package events

import (
	"context"
	"time"
)

type Publisher interface {
	PublishClickEvent(ctx context.Context, shortCode string) error
}

type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	ClickedAt time.Time `json:"clicked_at"`
}
