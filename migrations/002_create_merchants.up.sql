CREATE TABLE merchants (
    merchant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    current_chargeback_ratio NUMERIC(8,4) NOT NULL DEFAULT 0,
    total_transactions_30d INT NOT NULL DEFAULT 0,
    total_chargebacks_30d INT NOT NULL DEFAULT 0,
    is_flagged_by_processor BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_merchants_email ON merchants(email);
