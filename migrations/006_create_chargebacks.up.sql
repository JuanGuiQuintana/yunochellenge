CREATE TABLE chargebacks (
    chargeback_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processor_chargeback_id TEXT NOT NULL,
    processor_id UUID NOT NULL REFERENCES processors(processor_id),
    merchant_id UUID NOT NULL REFERENCES merchants(merchant_id),
    transaction_id TEXT NOT NULL,
    cardholder_id UUID NOT NULL REFERENCES cardholders(cardholder_id),
    reason_code_id UUID NOT NULL REFERENCES reason_codes(reason_code_id),
    amount BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    amount_usd NUMERIC(12,2),
    transaction_date TIMESTAMPTZ NOT NULL,
    notification_date TIMESTAMPTZ NOT NULL,
    dispute_deadline TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','under_review','won','lost','expired')),
    raw_payload JSONB NOT NULL DEFAULT '{}',
    risk_score INT NOT NULL DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    score_breakdown JSONB NOT NULL DEFAULT '{}',
    flags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_chargebacks_processor UNIQUE (processor_id, processor_chargeback_id)
);

CREATE INDEX idx_chargebacks_merchant_deadline ON chargebacks(merchant_id, dispute_deadline);
CREATE INDEX idx_chargebacks_merchant_score ON chargebacks(merchant_id, risk_score DESC);
CREATE INDEX idx_chargebacks_cardholder_notification ON chargebacks(cardholder_id, notification_date);
CREATE INDEX idx_chargebacks_merchant_rc_notification ON chargebacks(merchant_id, reason_code_id, notification_date);
CREATE INDEX idx_chargebacks_status ON chargebacks(status);
CREATE INDEX idx_chargebacks_notification_date ON chargebacks(notification_date);
