-- This schema file provides table definitions for sqlc to generate type-safe Go code
-- It doesn't need to include indexes, constraints, or triggers as those are handled by migrations

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    plan_type VARCHAR(50),
    billing_customer_id VARCHAR(255),
    is_active BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100),
    avatar_url TEXT,
    auth_id VARCHAR(255) UNIQUE,
    is_active BOOLEAN,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    first_run_complete BOOLEAN NOT NULL DEFAULT FALSE
);

-- Organization members table
CREATE TABLE organization_members (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL,
    is_primary_organization BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT unique_org_user UNIQUE (organization_id, user_id)
);
