CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users
(
    id             SERIAL PRIMARY KEY,
    uuid           UUID        NOT NULL                                     DEFAULT gen_random_uuid(),
    email          TEXT UNIQUE NOT NULL,
    password_hash  TEXT,
    role           TEXT        NOT NULL CHECK (role IN ('admin', 'worker')) DEFAULT 'worker',
    name           TEXT        NOT NULL,
    lastname       TEXT        NOT NULL,
    birth_date     DATE        NOT NULL,
    phone          TEXT        NOT NULL,

    created_at     TIMESTAMPTZ NOT NULL                                     DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL                                     DEFAULT now(),

    deleted_at     TIMESTAMPTZ,
    deleted_reason TEXT        NOT NULL,
    deleted_by     INTEGER     REFERENCES users (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS users_uuid_unique_idx
    ON users (uuid);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_active_idx
    ON users (lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS user_files
(
    id           SERIAL PRIMARY KEY,
    uuid         UUID        NOT NULL DEFAULT gen_random_uuid(),
    user_id      INTEGER     NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    bucket       TEXT        NOT NULL,
    storage_key  TEXT        NOT NULL, -- (S3/MinIO/FS)
    file_name    TEXT        NOT NULL,
    mime_type    TEXT        NOT NULL,
    size_bytes   BIGINT      NOT NULL CHECK (size_bytes >= 0),
    download_url TEXT        NOT NULL,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS user_files_uuid_unique_idx
    ON user_files (uuid);

-- considering we have existing admin user
INSERT INTO users (email,
                   password_hash,
                   role,
                   name,
                   lastname,
                   birth_date,
                   phone,
                   deleted_reason)
VALUES ('admin@example.com',
        crypt('admin123', gen_salt('bf', 12)),
        'admin',
        'Admin',
        'User',
        DATE '1990-01-01',
        '+0000000000',
        ''
       )