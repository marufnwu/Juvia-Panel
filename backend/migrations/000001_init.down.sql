-- Rollback: 000001_init
-- Description: Rollback initial database schema

-- Drop triggers first
DROP TRIGGER IF EXISTS update_ssl_certificates_updated_at ON ssl_certificates;
DROP TRIGGER IF EXISTS update_cron_jobs_updated_at ON cron_jobs;
DROP TRIGGER IF EXISTS update_services_updated_at ON services;
DROP TRIGGER IF EXISTS update_apps_updated_at ON apps;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS firewall_rules;
DROP TABLE IF EXISTS cron_jobs;
DROP TABLE IF EXISTS volumes;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS ssl_certificates;
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS backups;
DROP TABLE IF EXISTS service_connections;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS "pgcrypto";
