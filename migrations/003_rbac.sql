CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(20) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(20) NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    permission_name VARCHAR(50) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

INSERT INTO roles (name) VALUES ('admin'), ('staff'), ('user')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (name) VALUES
    ('user:list'),
    ('user:read:any'),
    ('user:update:any'),
    ('user:delete'),
    ('role:assign')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
SELECT 'admin', name FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES ('staff', 'user:list'), ('staff', 'user:read:any')
ON CONFLICT DO NOTHING;

UPDATE users SET role = 'user'
WHERE role IS NULL OR role NOT IN ('admin', 'staff', 'user');

ALTER TABLE users
    ALTER COLUMN role SET DEFAULT 'user',
    ALTER COLUMN role SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_role_fkey'
    ) THEN
        ALTER TABLE users
        ADD CONSTRAINT users_role_fkey
        FOREIGN KEY (role) REFERENCES roles(name);
    END IF;
END $$;
