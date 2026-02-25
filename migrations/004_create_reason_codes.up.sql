CREATE TABLE reason_codes (
    reason_code_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processor_id UUID NOT NULL REFERENCES processors(processor_id),
    raw_code TEXT NOT NULL,
    normalized_code TEXT NOT NULL,
    category TEXT NOT NULL,
    risk_level INT NOT NULL CHECK (risk_level BETWEEN 1 AND 5),
    typical_win_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reason_codes_processor_raw UNIQUE (processor_id, raw_code)
);
CREATE INDEX idx_reason_codes_normalized ON reason_codes(normalized_code);
