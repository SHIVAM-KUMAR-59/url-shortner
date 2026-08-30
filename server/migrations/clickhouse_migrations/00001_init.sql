-- +goose Up

CREATE TABLE click_events (
    short_code String,
    clicked_at DateTime64(3),
    ip_address String,
    user_agent String,
    referer String
) ENGINE = MergeTree()
ORDER BY (short_code, clicked_at);

-- +goose Down

DROP TABLE IF EXISTS click_events;