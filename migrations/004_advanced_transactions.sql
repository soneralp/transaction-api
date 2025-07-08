CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS scheduled_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    description TEXT,
    reference_id VARCHAR(100),
    to_user_id UUID,
    scheduled_at TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    recurring_type VARCHAR(20),
    recurring_config JSONB,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_transactions_user_id ON scheduled_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_transactions_scheduled_at ON scheduled_transactions(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_transactions_status ON scheduled_transactions(status);
CREATE INDEX IF NOT EXISTS idx_scheduled_transactions_pending ON scheduled_transactions(status, scheduled_at) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS batch_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(19,4) NOT NULL,
    item_count INTEGER NOT NULL,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_batch_transactions_user_id ON batch_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_batch_transactions_status ON batch_transactions(status);
CREATE INDEX IF NOT EXISTS idx_batch_transactions_created_at ON batch_transactions(created_at);

CREATE TABLE IF NOT EXISTS batch_transaction_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    batch_id UUID NOT NULL,
    transaction_id UUID NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    description TEXT,
    reference_id VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (batch_id) REFERENCES batch_transactions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_batch_transaction_items_batch_id ON batch_transaction_items(batch_id);
CREATE INDEX IF NOT EXISTS idx_batch_transaction_items_status ON batch_transaction_items(status);
CREATE INDEX IF NOT EXISTS idx_batch_transaction_items_transaction_id ON batch_transaction_items(transaction_id);

CREATE TABLE IF NOT EXISTS transaction_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    currency VARCHAR(3) NOT NULL,
    daily_limit DECIMAL(19,4) NOT NULL,
    weekly_limit DECIMAL(19,4) NOT NULL,
    monthly_limit DECIMAL(19,4) NOT NULL,
    single_limit DECIMAL(19,4) NOT NULL,
    daily_count INTEGER NOT NULL DEFAULT 0,
    weekly_count INTEGER NOT NULL DEFAULT 0,
    monthly_count INTEGER NOT NULL DEFAULT 0,
    daily_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    weekly_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    monthly_amount DECIMAL(19,4) NOT NULL DEFAULT 0,
    last_reset_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, currency)
);

CREATE INDEX IF NOT EXISTS idx_transaction_limits_user_id ON transaction_limits(user_id);
CREATE INDEX IF NOT EXISTS idx_transaction_limits_currency ON transaction_limits(currency);
CREATE INDEX IF NOT EXISTS idx_transaction_limits_user_currency ON transaction_limits(user_id, currency);

CREATE TABLE IF NOT EXISTS multi_currency_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    currency VARCHAR(3) NOT NULL,
    amount DECIMAL(19,4) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, currency)
);

CREATE INDEX IF NOT EXISTS idx_multi_currency_balances_user_id ON multi_currency_balances(user_id);
CREATE INDEX IF NOT EXISTS idx_multi_currency_balances_currency ON multi_currency_balances(currency);
CREATE INDEX IF NOT EXISTS idx_multi_currency_balances_user_currency ON multi_currency_balances(user_id, currency);

CREATE TABLE IF NOT EXISTS exchange_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_currency VARCHAR(3) NOT NULL,
    to_currency VARCHAR(3) NOT NULL,
    rate DECIMAL(19,6) NOT NULL,
    last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_currency, to_currency)
);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_from_currency ON exchange_rates(from_currency);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_to_currency ON exchange_rates(to_currency);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_currency_pair ON exchange_rates(from_currency, to_currency);

INSERT INTO exchange_rates (from_currency, to_currency, rate, source) VALUES
('USD', 'EUR', 0.85, 'default'),
('EUR', 'USD', 1.18, 'default'),
('USD', 'TRY', 8.50, 'default'),
('TRY', 'USD', 0.12, 'default'),
('USD', 'GBP', 0.73, 'default'),
('GBP', 'USD', 1.37, 'default'),
('EUR', 'TRY', 10.00, 'default'),
('TRY', 'EUR', 0.10, 'default'),
('EUR', 'GBP', 0.86, 'default'),
('GBP', 'EUR', 1.16, 'default'),
('GBP', 'TRY', 11.64, 'default'),
('TRY', 'GBP', 0.086, 'default')
ON CONFLICT (from_currency, to_currency) DO NOTHING;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'transactions' AND column_name = 'currency') THEN
        ALTER TABLE transactions ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'USD';
        CREATE INDEX IF NOT EXISTS idx_transactions_currency ON transactions(currency);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'balances' AND column_name = 'currency') THEN
        ALTER TABLE balances ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'USD';
        CREATE INDEX IF NOT EXISTS idx_balances_currency ON balances(currency);
    END IF;
END $$;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scheduled_transactions_updated_at BEFORE UPDATE ON scheduled_transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_batch_transactions_updated_at BEFORE UPDATE ON batch_transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_batch_transaction_items_updated_at BEFORE UPDATE ON batch_transaction_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transaction_limits_updated_at BEFORE UPDATE ON transaction_limits FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_multi_currency_balances_updated_at BEFORE UPDATE ON multi_currency_balances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_exchange_rates_updated_at BEFORE UPDATE ON exchange_rates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION reset_daily_transaction_limits()
RETURNS void AS $$
BEGIN
    UPDATE transaction_limits 
    SET daily_amount = 0, 
        daily_count = 0, 
        last_reset_date = CURRENT_TIMESTAMP
    WHERE DATE(last_reset_date) < CURRENT_DATE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION reset_weekly_transaction_limits()
RETURNS void AS $$
BEGIN
    UPDATE transaction_limits 
    SET weekly_amount = 0, 
        weekly_count = 0
    WHERE EXTRACT(WEEK FROM last_reset_date) < EXTRACT(WEEK FROM CURRENT_DATE)
       OR EXTRACT(YEAR FROM last_reset_date) < EXTRACT(YEAR FROM CURRENT_DATE);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION reset_monthly_transaction_limits()
RETURNS void AS $$
BEGIN
    UPDATE transaction_limits 
    SET monthly_amount = 0, 
        monthly_count = 0
    WHERE EXTRACT(MONTH FROM last_reset_date) < EXTRACT(MONTH FROM CURRENT_DATE)
       OR EXTRACT(YEAR FROM last_reset_date) < EXTRACT(YEAR FROM CURRENT_DATE);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW pending_scheduled_transactions AS
SELECT * FROM scheduled_transactions 
WHERE status = 'pending' 
  AND scheduled_at <= CURRENT_TIMESTAMP
ORDER BY scheduled_at ASC;

CREATE OR REPLACE VIEW batch_transaction_stats AS
SELECT 
    bt.id,
    bt.user_id,
    bt.type,
    bt.status,
    bt.total_amount,
    bt.item_count,
    COUNT(bti.id) as processed_items,
    COUNT(CASE WHEN bti.status = 'completed' THEN 1 END) as completed_items,
    COUNT(CASE WHEN bti.status = 'failed' THEN 1 END) as failed_items,
    bt.created_at,
    bt.processed_at
FROM batch_transactions bt
LEFT JOIN batch_transaction_items bti ON bt.id = bti.batch_id
GROUP BY bt.id, bt.user_id, bt.type, bt.status, bt.total_amount, bt.item_count, bt.created_at, bt.processed_at;

CREATE OR REPLACE VIEW transaction_limit_usage AS
SELECT 
    tl.user_id,
    tl.currency,
    tl.daily_limit,
    tl.daily_amount,
    tl.daily_count,
    tl.weekly_limit,
    tl.weekly_amount,
    tl.weekly_count,
    tl.monthly_limit,
    tl.monthly_amount,
    tl.monthly_count,
    tl.single_limit,
    ROUND((tl.daily_amount / tl.daily_limit) * 100, 2) as daily_usage_percent,
    ROUND((tl.weekly_amount / tl.weekly_limit) * 100, 2) as weekly_usage_percent,
    ROUND((tl.monthly_amount / tl.monthly_limit) * 100, 2) as monthly_usage_percent,
    tl.last_reset_date,
    tl.is_active
FROM transaction_limits tl;
