-- Enable Row Level Security on tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_members ENABLE ROW LEVEL SECURITY;

-- Create a function to check if the current user has access to an organization
CREATE OR REPLACE FUNCTION public.user_is_organization_member(org_id uuid)
RETURNS boolean AS $$
DECLARE
    current_auth_id text;
    user_id uuid;
    found boolean;
BEGIN
    -- Get current user's auth ID from Supabase auth.uid()
    current_auth_id := auth.uid()::text;
    IF current_auth_id IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- Get the user ID from the auth ID
    SELECT id INTO user_id FROM users WHERE auth_id = current_auth_id;
    IF user_id IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- Check if the user is a member of the organization
    SELECT EXISTS (
        SELECT 1 FROM organization_members 
        WHERE organization_id = org_id AND user_id = user_id
    ) INTO found;
    
    RETURN found;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- RLS policy for users: users can only see and modify their own data
CREATE POLICY users_self_access ON users
    USING (auth_id = auth.uid()::text)
    WITH CHECK (auth_id = auth.uid()::text);

-- Allow all users to see organization details if they are a member
CREATE POLICY organizations_member_select ON organizations
    FOR SELECT
    USING (public.user_is_organization_member(id));

-- Only organization owners or admins can modify organization details
CREATE POLICY organizations_owner_or_admin_update ON organizations
    FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM organization_members om
            JOIN users u ON u.id = om.user_id
            WHERE om.organization_id = organizations.id
            AND u.auth_id = auth.uid()::text
            AND om.role IN ('owner', 'admin')
        )
    );

-- Users can see organization members if they are a member of the organization
CREATE POLICY org_members_select ON organization_members
    FOR SELECT
    USING (public.user_is_organization_member(organization_id));

-- Only organization owners can add/remove members
CREATE POLICY org_members_owner_modify ON organization_members
    USING (
        EXISTS (
            SELECT 1 FROM organization_members om
            JOIN users u ON u.id = om.user_id
            WHERE om.organization_id = organization_members.organization_id
            AND u.auth_id = auth.uid()::text
            AND om.role = 'owner'
        )
    );

-- Allow service role to bypass RLS
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE organizations FORCE ROW LEVEL SECURITY;
ALTER TABLE organization_members FORCE ROW LEVEL SECURITY;

-- Create policy to allow the service role to bypass RLS (for backend operations).
-- Scoped TO service_role so it does NOT apply to anon/authenticated — otherwise
-- a permissive USING(true) for ALL roles would OR-override the self-access
-- policies and nullify RLS.
CREATE POLICY service_role_bypass ON users
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY service_role_bypass ON organizations
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY service_role_bypass ON organization_members
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);