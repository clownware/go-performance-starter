-- Drop all policies
DROP POLICY IF EXISTS service_role_bypass ON organization_members;
DROP POLICY IF EXISTS service_role_bypass ON organizations;
DROP POLICY IF EXISTS service_role_bypass ON users;

DROP POLICY IF EXISTS org_members_owner_modify ON organization_members;
DROP POLICY IF EXISTS org_members_select ON organization_members;
DROP POLICY IF EXISTS organizations_owner_or_admin_update ON organizations;
DROP POLICY IF EXISTS organizations_member_select ON organizations;
DROP POLICY IF EXISTS users_self_access ON users;

-- Drop the helper function
DROP FUNCTION IF EXISTS public.user_is_organization_member(uuid);

-- Disable RLS on all tables
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE organizations DISABLE ROW LEVEL SECURITY;
ALTER TABLE organization_members DISABLE ROW LEVEL SECURITY;