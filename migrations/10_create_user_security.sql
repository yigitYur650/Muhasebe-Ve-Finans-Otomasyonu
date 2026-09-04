-- Migration: 10_create_user_security.sql
-- Description: Table for storing security questions and bcrypt-hashed security answers for password recovery.

CREATE TABLE IF NOT EXISTS public.user_security (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    security_question TEXT NOT NULL,
    security_answer_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookup by email
CREATE INDEX IF NOT EXISTS idx_user_security_email ON public.user_security(email);

-- Enable & Force Row Level Security
ALTER TABLE public.user_security ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_security FORCE ROW LEVEL SECURITY;

-- RLS Policy: Users can view and update their own security settings
CREATE POLICY user_security_owner_policy ON public.user_security
    FOR ALL
    USING (user_id = auth.uid())
    WITH CHECK (user_id = auth.uid());
