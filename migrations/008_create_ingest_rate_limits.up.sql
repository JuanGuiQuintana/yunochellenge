CREATE TABLE ingest_rate_limits (
    processor_id UUID NOT NULL REFERENCES processors(processor_id),
    window_minute TIMESTAMPTZ NOT NULL,
    request_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (processor_id, window_minute)
);
