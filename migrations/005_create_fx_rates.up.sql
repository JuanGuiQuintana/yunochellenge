CREATE TABLE fx_rates (
    currency CHAR(3) NOT NULL,
    date DATE NOT NULL,
    rate_to_usd NUMERIC(18,8) NOT NULL,
    source TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (currency, date)
);
