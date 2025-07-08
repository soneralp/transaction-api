CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS ha_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component VARCHAR(50) NOT NULL, -- database, loadbalancer, circuitbreaker, fallback
    config_key VARCHAR(100) NOT NULL,
    config_value TEXT NOT NULL,
    config_type VARCHAR(20) NOT NULL DEFAULT 'string', -- string, integer, boolean, json
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(component, config_key)
);

CREATE TABLE IF NOT EXISTS ha_health_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component VARCHAR(50) NOT NULL,
    component_id VARCHAR(100),
    status VARCHAR(20) NOT NULL, -- healthy, unhealthy, degraded
    latency_ms INTEGER,
    error_message TEXT,
    metadata JSONB,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ha_circuit_breaker_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    breaker_name VARCHAR(100) NOT NULL,
    state VARCHAR(20) NOT NULL, -- CLOSED, OPEN, HALF_OPEN
    requests_count INTEGER NOT NULL DEFAULT 0,
    errors_count INTEGER NOT NULL DEFAULT 0,
    consecutive_errors INTEGER NOT NULL DEFAULT 0,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,4),
    last_error TEXT,
    state_changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ha_load_balancer_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    backend_id VARCHAR(100) NOT NULL,
    backend_url VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL, -- active, inactive, unhealthy
    health_score DECIMAL(3,2), -- 0.00 - 1.00
    latency_ms INTEGER,
    weight INTEGER NOT NULL DEFAULT 1,
    last_check_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ha_fallback_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation_key VARCHAR(100) NOT NULL,
    strategy VARCHAR(20) NOT NULL, -- sequential, parallel, degradation
    primary_success BOOLEAN NOT NULL,
    fallback_used BOOLEAN NOT NULL DEFAULT FALSE,
    fallback_level INTEGER, -- 1, 2, 3, etc.
    cache_hit BOOLEAN NOT NULL DEFAULT FALSE,
    execution_time_ms INTEGER,
    error_message TEXT,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ha_database_replication_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_name VARCHAR(100) NOT NULL,
    node_role VARCHAR(20) NOT NULL, -- master, slave, read_replica
    node_host VARCHAR(255) NOT NULL,
    node_port INTEGER NOT NULL,
    replication_lag_seconds INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync_at TIMESTAMP,
    health_status VARCHAR(20) NOT NULL DEFAULT 'unknown', -- healthy, unhealthy, unknown
    latency_ms INTEGER,
    last_check_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(node_name)
);

CREATE TABLE IF NOT EXISTS ha_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,4) NOT NULL,
    metric_unit VARCHAR(20), -- requests, ms, percentage, count
    component VARCHAR(50) NOT NULL,
    component_id VARCHAR(100),
    tags JSONB,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ha_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_type VARCHAR(50) NOT NULL, -- circuit_breaker_open, database_down, load_balancer_unhealthy
    severity VARCHAR(20) NOT NULL, -- low, medium, high, critical
    component VARCHAR(50) NOT NULL,
    component_id VARCHAR(100),
    message TEXT NOT NULL,
    metadata JSONB,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ha_configurations_component ON ha_configurations(component);
CREATE INDEX IF NOT EXISTS idx_ha_configurations_active ON ha_configurations(is_active);

CREATE INDEX IF NOT EXISTS idx_ha_health_history_component ON ha_health_history(component);
CREATE INDEX IF NOT EXISTS idx_ha_health_history_status ON ha_health_history(status);
CREATE INDEX IF NOT EXISTS idx_ha_health_history_checked_at ON ha_health_history(checked_at);

CREATE INDEX IF NOT EXISTS idx_ha_circuit_breaker_history_name ON ha_circuit_breaker_history(breaker_name);
CREATE INDEX IF NOT EXISTS idx_ha_circuit_breaker_history_state ON ha_circuit_breaker_history(state);
CREATE INDEX IF NOT EXISTS idx_ha_circuit_breaker_history_state_changed_at ON ha_circuit_breaker_history(state_changed_at);

CREATE INDEX IF NOT EXISTS idx_ha_load_balancer_history_backend_id ON ha_load_balancer_history(backend_id);
CREATE INDEX IF NOT EXISTS idx_ha_load_balancer_history_status ON ha_load_balancer_history(status);
CREATE INDEX IF NOT EXISTS idx_ha_load_balancer_history_last_check_at ON ha_load_balancer_history(last_check_at);

CREATE INDEX IF NOT EXISTS idx_ha_fallback_history_operation_key ON ha_fallback_history(operation_key);
CREATE INDEX IF NOT EXISTS idx_ha_fallback_history_strategy ON ha_fallback_history(strategy);
CREATE INDEX IF NOT EXISTS idx_ha_fallback_history_executed_at ON ha_fallback_history(executed_at);

CREATE INDEX IF NOT EXISTS idx_ha_database_replication_status_node_name ON ha_database_replication_status(node_name);
CREATE INDEX IF NOT EXISTS idx_ha_database_replication_status_role ON ha_database_replication_status(node_role);
CREATE INDEX IF NOT EXISTS idx_ha_database_replication_status_health ON ha_database_replication_status(health_status);

CREATE INDEX IF NOT EXISTS idx_ha_metrics_name ON ha_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_ha_metrics_component ON ha_metrics(component);
CREATE INDEX IF NOT EXISTS idx_ha_metrics_recorded_at ON ha_metrics(recorded_at);

CREATE INDEX IF NOT EXISTS idx_ha_alerts_type ON ha_alerts(alert_type);
CREATE INDEX IF NOT EXISTS idx_ha_alerts_severity ON ha_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_ha_alerts_component ON ha_alerts(component);
CREATE INDEX IF NOT EXISTS idx_ha_alerts_resolved ON ha_alerts(is_resolved);
CREATE INDEX IF NOT EXISTS idx_ha_alerts_created_at ON ha_alerts(created_at);

INSERT INTO ha_configurations (component, config_key, config_value, config_type) VALUES
('database', 'replication_enabled', 'true', 'boolean'),
('database', 'failover_enabled', 'true', 'boolean'),
('database', 'health_check_interval', '30', 'integer'),
('database', 'max_connections', '100', 'integer'),
('database', 'max_idle_connections', '10', 'integer'),
('database', 'connection_lifetime', '3600', 'integer'),

('loadbalancer', 'strategy', 'round_robin', 'string'),
('loadbalancer', 'health_check_interval', '30', 'integer'),
('loadbalancer', 'health_check_timeout', '5', 'integer'),
('loadbalancer', 'max_retries', '3', 'integer'),

('circuitbreaker', 'failure_threshold', '5', 'integer'),
('circuitbreaker', 'success_threshold', '3', 'integer'),
('circuitbreaker', 'timeout', '60', 'integer'),
('circuitbreaker', 'half_open_max_requests', '3', 'integer'),
('circuitbreaker', 'min_request_count', '10', 'integer'),

('fallback', 'max_retries', '3', 'integer'),
('fallback', 'retry_delay', '1000', 'integer'),
('fallback', 'timeout', '30000', 'integer'),
('fallback', 'enable_caching', 'true', 'boolean'),
('fallback', 'cache_ttl', '300', 'integer'),
('fallback', 'enable_degradation', 'true', 'boolean')
ON CONFLICT (component, config_key) DO NOTHING;

CREATE OR REPLACE FUNCTION update_ha_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_ha_configurations_updated_at BEFORE UPDATE ON ha_configurations FOR EACH ROW EXECUTE FUNCTION update_ha_updated_at_column();
CREATE TRIGGER update_ha_database_replication_status_updated_at BEFORE UPDATE ON ha_database_replication_status FOR EACH ROW EXECUTE FUNCTION update_ha_updated_at_column();
CREATE TRIGGER update_ha_alerts_updated_at BEFORE UPDATE ON ha_alerts FOR EACH ROW EXECUTE FUNCTION update_ha_updated_at_column();

CREATE OR REPLACE FUNCTION cleanup_old_ha_health_history()
RETURNS void AS $$
BEGIN
    DELETE FROM ha_health_history WHERE checked_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
    
    DELETE FROM ha_circuit_breaker_history WHERE state_changed_at < CURRENT_TIMESTAMP - INTERVAL '7 days';
    
    DELETE FROM ha_load_balancer_history WHERE last_check_at < CURRENT_TIMESTAMP - INTERVAL '7 days';
    
    DELETE FROM ha_fallback_history WHERE executed_at < CURRENT_TIMESTAMP - INTERVAL '7 days';
    
    DELETE FROM ha_metrics WHERE recorded_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
    
    DELETE FROM ha_alerts WHERE is_resolved = true AND resolved_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_ha_system_status()
RETURNS TABLE(
    component VARCHAR(50),
    status VARCHAR(20),
    active_count INTEGER,
    total_count INTEGER,
    last_check TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'database'::VARCHAR(50) as component,
        CASE 
            WHEN COUNT(*) FILTER (WHERE health_status = 'healthy') = COUNT(*) THEN 'healthy'
            WHEN COUNT(*) FILTER (WHERE health_status = 'unhealthy') = COUNT(*) THEN 'unhealthy'
            ELSE 'degraded'
        END::VARCHAR(20) as status,
        COUNT(*) FILTER (WHERE is_active = true)::INTEGER as active_count,
        COUNT(*)::INTEGER as total_count,
        MAX(last_check_at)::TIMESTAMP as last_check
    FROM ha_database_replication_status
    
    UNION ALL
    
    SELECT 
        'loadbalancer'::VARCHAR(50) as component,
        CASE 
            WHEN COUNT(*) FILTER (WHERE status = 'active') = COUNT(*) THEN 'healthy'
            WHEN COUNT(*) FILTER (WHERE status = 'inactive') = COUNT(*) THEN 'unhealthy'
            ELSE 'degraded'
        END::VARCHAR(20) as status,
        COUNT(*) FILTER (WHERE status = 'active')::INTEGER as active_count,
        COUNT(*)::INTEGER as total_count,
        MAX(last_check_at)::TIMESTAMP as last_check
    FROM ha_load_balancer_history;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE VIEW ha_dashboard AS
SELECT 
    'system_overview' as section,
    json_build_object(
        'database_nodes', (SELECT COUNT(*) FROM ha_database_replication_status WHERE is_active = true),
        'load_balancer_backends', (SELECT COUNT(*) FROM ha_load_balancer_history WHERE status = 'active'),
        'circuit_breakers', (SELECT COUNT(DISTINCT breaker_name) FROM ha_circuit_breaker_history),
        'active_alerts', (SELECT COUNT(*) FROM ha_alerts WHERE is_resolved = false)
    ) as data

UNION ALL

SELECT 
    'recent_alerts' as section,
    json_build_object(
        'alerts', (
            SELECT json_agg(
                json_build_object(
                    'id', id,
                    'type', alert_type,
                    'severity', severity,
                    'message', message,
                    'created_at', created_at
                )
            )
            FROM ha_alerts 
            WHERE is_resolved = false 
            ORDER BY created_at DESC 
            LIMIT 10
        )
    ) as data

UNION ALL

SELECT 
    'performance_metrics' as section,
    json_build_object(
        'avg_latency', (
            SELECT AVG(metric_value) 
            FROM ha_metrics 
            WHERE metric_name = 'latency' 
            AND recorded_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
        ),
        'error_rate', (
            SELECT AVG(metric_value) 
            FROM ha_metrics 
            WHERE metric_name = 'error_rate' 
            AND recorded_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
        ),
        'request_count', (
            SELECT SUM(metric_value) 
            FROM ha_metrics 
            WHERE metric_name = 'request_count' 
            AND recorded_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
        )
    ) as data;
