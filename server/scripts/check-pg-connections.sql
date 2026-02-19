-- Check PostgreSQL Connection Pool Settings
-- Run this with: psql $DATABASE_URL -f check-pg-connections.sql

\echo '======================================'
\echo 'PostgreSQL Connection Pool Check'
\echo '======================================'
\echo ''

\echo 'Current max_connections setting:'
SHOW max_connections;
\echo ''

\echo 'Current active connections:'
SELECT count(*) as active_connections FROM pg_stat_activity;
\echo ''

\echo 'Connections by state:'
SELECT
    state,
    count(*) as count
FROM pg_stat_activity
GROUP BY state;
\echo ''

\echo 'Connections by application:'
SELECT
    application_name,
    count(*) as count
FROM pg_stat_activity
WHERE application_name != ''
GROUP BY application_name;
\echo ''

\echo 'Check if restart needed:'
SELECT
    name,
    setting as current_value,
    unit,
    pending_restart,
    CASE
        WHEN pending_restart THEN 'RESTART REQUIRED'
        ELSE 'No restart needed'
    END as status
FROM pg_settings
WHERE name = 'max_connections';
\echo ''

\echo 'Recommendation based on your app:'
\echo '  - Go app MaxOpenConns: 200'
\echo '  - Recommended PostgreSQL max_connections: >= 250'
\echo '  - Current Railway default: usually 100-200'
