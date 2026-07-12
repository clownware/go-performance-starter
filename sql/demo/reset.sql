-- Nightly demo reset (ADR-031): clear the content anonymous guests have
-- accumulated so the public demo cannot be permanently vandalized or grow
-- without bound. Run via `task demo:reset`, which refuses without DEMO_MODE=1.
--
-- Deliberately NOT deleted here:
--   * users rows — the TTL reaper (ADR-024) owns identity deletion end-to-end
--     (GoTrue admin API + public row). Deleting public rows here would orphan
--     the auth-side records the reaper finds through them.
--   * registered (non-anonymous) users' data — upgraded accounts keep their
--     progress; that's the promise of the upgrade flow.
--   * quiz_questions — read-only under RLS; the seed re-asserts them anyway.

BEGIN;

DELETE FROM flashcards
WHERE user_id IN (SELECT id FROM users WHERE is_anonymous);

DELETE FROM quiz_attempts
WHERE user_id IN (SELECT id FROM users WHERE is_anonymous);

COMMIT;
