CREATE TABLE cardholders (
    cardholder_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bin CHAR(6) NOT NULL,
    last4 CHAR(4) NOT NULL,
    card_fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cardholders_fingerprint UNIQUE (card_fingerprint)
);
