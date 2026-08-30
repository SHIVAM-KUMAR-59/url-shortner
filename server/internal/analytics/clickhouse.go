package analytics

import (
	"context"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseReader struct {
	conn clickhouse.Conn
}

func NewClickHouseReader(conn clickhouse.Conn) *ClickHouseReader {
	return &ClickHouseReader{
		conn: conn,
	}
}

func (r *ClickHouseReader) GetStats(
	ctx context.Context,
	shortCode string,
) (int64, *time.Time, error) {
	var totalClicks uint64
	var lastClicked *time.Time

	row := r.conn.QueryRow(
		ctx,
		`SELECT COUNT(*), MAX(clicked_at)
		 FROM click_events
		 WHERE short_code = ?`,
		shortCode,
	)

	if err := row.Scan(&totalClicks, &lastClicked); err != nil {
		log.Printf("ClickHouse GetStats failed for %s: %v", shortCode, err)
		return 0, nil, err
	}

	return int64(totalClicks), lastClicked, nil
}
