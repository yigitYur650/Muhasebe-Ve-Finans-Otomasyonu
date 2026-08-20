-- 02_create_tenant_members.sql
-- Amaç: auth.users ile tenants arasında rol bazlı üyelik.

CREATE TABLE public.tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'muhasebeci', 'standart')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);

ALTER TABLE public.tenant_members ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Kullanıcı kendi üyeliklerini görür"
ON public.tenant_members FOR SELECT
TO authenticated
USING (user_id = auth.uid());
