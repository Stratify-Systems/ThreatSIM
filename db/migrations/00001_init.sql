-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE simulations (
    id VARCHAR(255) PRIMARY KEY,
    plugin_id VARCHAR(100) NOT NULL,
    target VARCHAR(255) NOT NULL,
    events_num INT DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration VARCHAR(50)
);

CREATE TABLE events (
    id VARCHAR(255) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    source_ip VARCHAR(100) NOT NULL,
    target VARCHAR(255) NOT NULL,
    service VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    plugin_id VARCHAR(100) NOT NULL
);

CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    source_ip VARCHAR(100) NOT NULL,
    score INT NOT NULL,
    threat_level VARCHAR(50) NOT NULL,
    factors JSONB,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE alerts;
DROP TABLE events;
DROP TABLE simulations;
