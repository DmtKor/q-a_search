-- Module 00 — Contracts: initial schema
-- PostgreSQL + pgvector + FTS. Idempotent: safe to re-run (IF NOT EXISTS).

-- Enable pgvector for case_embeddings (cosine similarity)
CREATE EXTENSION IF NOT EXISTS vector;

-- =============================================================================
-- apps (clients/applications + JSON settings)
-- =============================================================================
CREATE TABLE IF NOT EXISTS apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_apps_name ON apps(name);

-- =============================================================================
-- auth_tokens (Bearer: app | staff)
-- =============================================================================
CREATE TABLE IF NOT EXISTS auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL,
    token_type VARCHAR(20) NOT NULL CHECK (token_type IN ('app', 'staff')),
    role VARCHAR(50),
    app_id UUID REFERENCES apps(id) ON DELETE SET NULL,
    label TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_token_hash ON auth_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_type ON auth_tokens(token_type);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_app_id ON auth_tokens(app_id);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_active_expires ON auth_tokens(is_active, expires_at) WHERE is_active = TRUE;

-- =============================================================================
-- cases (knowledge base; FTS via search_tsv)
-- =============================================================================
CREATE TABLE IF NOT EXISTS cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(100) NOT NULL,
    title TEXT NOT NULL,
    questions JSONB NOT NULL DEFAULT '[]'::jsonb,
    keywords JSONB DEFAULT '[]'::jsonb,
    response_template TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'pending_review', 'approved', 'archived')),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    notes TEXT,
    search_tsv tsvector
);

CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status);
CREATE INDEX IF NOT EXISTS idx_cases_category ON cases(category);
CREATE INDEX IF NOT EXISTS idx_cases_updated_at ON cases(updated_at);
CREATE INDEX IF NOT EXISTS idx_cases_created_by ON cases(created_by);

-- FTS: GIN index on search_tsv (filled by app on INSERT/UPDATE)
CREATE INDEX IF NOT EXISTS idx_cases_search_tsv ON cases USING GIN(search_tsv);

COMMENT ON COLUMN cases.search_tsv IS 'Full-text search vector (title, keywords, questions). Filled by application.';

-- =============================================================================
-- case_embeddings (pgvector; one row per case, only for status = approved)
-- Application invariant: insert/update only when case.status = 'approved'; delete when leaving approved.
-- =============================================================================
CREATE TABLE IF NOT EXISTS case_embeddings (
    case_id UUID PRIMARY KEY REFERENCES cases(id) ON DELETE CASCADE,
    embedding vector(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cosine similarity index (vector_cosine_ops for ANN search)
CREATE INDEX IF NOT EXISTS idx_case_embeddings_hnsw ON case_embeddings
    USING hnsw (embedding vector_cosine_ops);

COMMENT ON TABLE case_embeddings IS 'One embedding per case. Only for cases.status = approved.';

-- =============================================================================
-- tickets (low-confidence queries; optional convert-to-case)
-- =============================================================================
CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    category VARCHAR(100),
    confidence DOUBLE PRECISION,
    status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    assigned_to VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,
    converted_to_case_id UUID REFERENCES cases(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at);
CREATE INDEX IF NOT EXISTS idx_tickets_category ON tickets(category);

-- =============================================================================
-- request_metrics (no query text stored)
-- =============================================================================
CREATE TABLE IF NOT EXISTS request_metrics (
    id BIGSERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,
    status_code INT NOT NULL,
    response_time_ms INT NOT NULL,
    token_id UUID,
    app_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_request_metrics_created_at ON request_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_request_metrics_endpoint ON request_metrics(endpoint);
