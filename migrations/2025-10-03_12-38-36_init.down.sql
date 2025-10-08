DROP INDEX IF EXISTS users_email_unique_active_idx;
DROP INDEX IF EXISTS users_uuid_unique_idx;
DROP INDEX IF EXISTS user_files_uuid_unique_idx;
DROP TABLE IF EXISTS user_files;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS pgcrypto;