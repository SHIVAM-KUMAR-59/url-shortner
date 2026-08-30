package events

import (
	"context"
	"time"
)

type Publisher interface {
	PublishClickEvent(ctx context.Context, event ClickEvent) error
}

type ClickEvent struct {
	ShortCode  string    `json:"short_code"`
	ClickedAt  time.Time `json:"clicked_at"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	HTTPMethod string    `json:"http_method"`
	StatusCode int       `json:"status_code"`
}
