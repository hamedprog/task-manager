SELECT 'CREATE DATABASE taskmanager'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'taskmanager'
)\gexec

SELECT 'CREATE DATABASE taskmanager_test'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'taskmanager_test'
)\gexec
