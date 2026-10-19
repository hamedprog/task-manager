SELECT 'DROP DATABASE taskmanager'
WHERE EXISTS (
    SELECT FROM pg_database WHERE datname = 'taskmanager'
)\gexec

SELECT 'DROP DATABASE taskmanager_test'
WHERE EXISTS (
    SELECT FROM pg_database WHERE datname = 'taskmanager_test'
)\gexec