CREATE TABLE chargeback_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chargeback_id UUID NOT NULL REFERENCES chargebacks(chargeback_id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    actor TEXT NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cb_events_chargeback ON chargeback_events(chargeback_id, created_at);
