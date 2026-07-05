-- No-op: 000005 recreates the service_role_bypass policies in the same
-- (correctly scoped) form that 000002 now defines. Rolling back must NOT
-- restore the unscoped pre-fix policies — that would reopen the RLS bypass.
SELECT 1;
