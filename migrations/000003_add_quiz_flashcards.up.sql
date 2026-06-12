-- Demo domain (ADR-024): architecture quiz + saveable flashcards.
-- Ownership/RLS mirrors the users_self_access pattern in 000002 so anonymous
-- guests (a users row whose auth_id is the anonymous auth.uid()) are scoped
-- identically to registered users.

-- Quiz questions: seeded reference content, readable by everyone.
CREATE TABLE quiz_questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    topic VARCHAR(100) NOT NULL,
    prompt TEXT NOT NULL,
    choices JSONB NOT NULL,
    correct_index INTEGER NOT NULL,
    explanation TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_quiz_questions_topic ON quiz_questions(topic);

-- Quiz attempts: user-owned record of each answered question.
CREATE TABLE quiz_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    selected_index INTEGER NOT NULL,
    is_correct BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_quiz_attempts_user_id ON quiz_attempts(user_id);

-- Flashcards: user-owned, typically created from a wrong answer.
CREATE TABLE flashcards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id UUID REFERENCES quiz_questions(id) ON DELETE SET NULL,
    front TEXT NOT NULL,
    back TEXT NOT NULL,
    is_known BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_flashcards_user_id ON flashcards(user_id);

CREATE TRIGGER update_flashcards_updated_at
BEFORE UPDATE ON flashcards
FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Row Level Security
ALTER TABLE quiz_questions ENABLE ROW LEVEL SECURITY;
ALTER TABLE quiz_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE flashcards ENABLE ROW LEVEL SECURITY;

-- Quiz questions are public reference content (readable by anonymous guests too).
CREATE POLICY quiz_questions_public_read ON quiz_questions
    FOR SELECT
    USING (true);

-- Attempts and flashcards are visible/modifiable only by their owner, resolved
-- through users.auth_id = auth.uid() (same mechanism as 000002).
CREATE POLICY quiz_attempts_self_access ON quiz_attempts
    USING (EXISTS (SELECT 1 FROM users u WHERE u.id = quiz_attempts.user_id AND u.auth_id = auth.uid()::text))
    WITH CHECK (EXISTS (SELECT 1 FROM users u WHERE u.id = quiz_attempts.user_id AND u.auth_id = auth.uid()::text));

CREATE POLICY flashcards_self_access ON flashcards
    USING (EXISTS (SELECT 1 FROM users u WHERE u.id = flashcards.user_id AND u.auth_id = auth.uid()::text))
    WITH CHECK (EXISTS (SELECT 1 FROM users u WHERE u.id = flashcards.user_id AND u.auth_id = auth.uid()::text));

ALTER TABLE quiz_questions FORCE ROW LEVEL SECURITY;
ALTER TABLE quiz_attempts FORCE ROW LEVEL SECURITY;
ALTER TABLE flashcards FORCE ROW LEVEL SECURITY;

-- Service-role bypass for backend operations and seeding (mirrors 000002).
-- Scoped TO service_role so it doesn't nullify the self-access policies.
CREATE POLICY service_role_bypass ON quiz_questions FOR ALL TO service_role USING (true) WITH CHECK (true);
CREATE POLICY service_role_bypass ON quiz_attempts FOR ALL TO service_role USING (true) WITH CHECK (true);
CREATE POLICY service_role_bypass ON flashcards FOR ALL TO service_role USING (true) WITH CHECK (true);
