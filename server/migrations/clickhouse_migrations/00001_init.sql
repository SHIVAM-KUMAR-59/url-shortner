-- +goose Up

CREATE TABLE click_events (
    short_code String,
    clicked_at DateTime
) ENGINE = MergeTree()
ORDER BY (short_code, clicked_at);

-- +goose Down

DROP TABLE click_events;