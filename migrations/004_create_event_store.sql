CREATE TABLE IF NOT EXISTS event_store (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    version BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_store_aggregate_id ON event_store(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_store_type ON event_store(type);
CREATE INDEX IF NOT EXISTS idx_event_store_timestamp ON event_store(timestamp);
CREATE INDEX IF NOT EXISTS idx_event_store_aggregate_version ON event_store(aggregate_id, version);

CREATE INDEX IF NOT EXISTS idx_event_store_type_timestamp ON event_store(type, timestamp);

CREATE INDEX IF NOT EXISTS idx_event_store_recent ON event_store(timestamp) WHERE timestamp > NOW() - INTERVAL '30 days';

CREATE OR REPLACE FUNCTION get_aggregate_latest_version(agg_id UUID)
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE(
        (SELECT MAX(version) FROM event_store WHERE aggregate_id = agg_id),
        0
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION aggregate_exists(agg_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(SELECT 1 FROM event_store WHERE aggregate_id = agg_id LIMIT 1);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_aggregate_event_count(agg_id UUID)
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE(
        (SELECT COUNT(*) FROM event_store WHERE aggregate_id = agg_id),
        0
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW event_statistics AS
SELECT 
    type,
    COUNT(*) as event_count,
    COUNT(DISTINCT aggregate_id) as aggregate_count,
    MIN(timestamp) as first_event_time,
    MAX(timestamp) as last_event_time
FROM event_store
GROUP BY type
ORDER BY event_count DESC;

CREATE OR REPLACE VIEW aggregate_statistics AS
SELECT 
    aggregate_id,
    COUNT(*) as event_count,
    MIN(timestamp) as first_event_time,
    MAX(timestamp) as last_event_time,
    MAX(version) as current_version
FROM event_store
GROUP BY aggregate_id
ORDER BY event_count DESC;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW event_audit_log AS
SELECT 
    id,
    type,
    aggregate_id,
    version,
    timestamp,
    created_at,
    CASE 
        WHEN type LIKE 'transaction.%' THEN 'transaction'
        WHEN type LIKE 'balance.%' THEN 'balance'
        WHEN type LIKE 'user.%' THEN 'user'
        ELSE 'unknown'
    END as aggregate_type
FROM event_store
ORDER BY timestamp DESC;

CREATE MATERIALIZED VIEW IF NOT EXISTS event_summary_mv AS
SELECT 
    DATE_TRUNC('day', timestamp) as event_date,
    type,
    COUNT(*) as daily_event_count,
    COUNT(DISTINCT aggregate_id) as daily_aggregate_count
FROM event_store
GROUP BY DATE_TRUNC('day', timestamp), type
ORDER BY event_date DESC, daily_event_count DESC;

CREATE UNIQUE INDEX IF NOT EXISTS idx_event_summary_mv_date_type ON event_summary_mv(event_date, type);

CREATE OR REPLACE FUNCTION refresh_event_summary_mv()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY event_summary_mv;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_old_events(days_to_keep INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM event_store 
    WHERE timestamp < NOW() - INTERVAL '1 day' * days_to_keep;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql; 