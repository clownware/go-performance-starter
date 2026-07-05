-- Forward migration for existing deployments (issue #29).
--
-- Migration 000002 originally created the service_role_bypass policies
-- without a role scope (applying to ALL roles), which OR-overrode the
-- self-access policies and nullified RLS. The file was later fixed in place
-- (scoped TO service_role), but databases migrated before that fix still
-- carry the unscoped policies. Recreating the policies is idempotent: fresh
-- databases get identical policies, previously-deployed databases get the
-- fix.

DROP POLICY IF EXISTS service_role_bypass ON users;
DROP POLICY IF EXISTS service_role_bypass ON organizations;
DROP POLICY IF EXISTS service_role_bypass ON organization_members;

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
