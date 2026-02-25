CREATE TABLE merchant_ratio_snapshots (
    merchant_id UUID NOT NULL REFERENCES merchants(merchant_id),
    date DATE NOT NULL,
    chargeback_count INT NOT NULL DEFAULT 0,
    transaction_count INT NOT NULL DEFAULT 0,
    ratio NUMERIC(8,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (merchant_id, date)
);
