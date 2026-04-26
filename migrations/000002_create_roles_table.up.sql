CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name, description)
VALUES
    ('super_admin', 'Full system access'),
    ('admin', 'Administrative access'),
    ('user', 'Standard application user'),
    ('tenant_admin', 'Tenant administrator'),
    ('tenant_member', 'Tenant member')
ON CONFLICT (name) DO NOTHING;
