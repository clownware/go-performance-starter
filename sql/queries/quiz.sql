-- name: ListQuizQuestions :many
SELECT id, slug, topic, prompt, choices, correct_index, explanation, created_at
FROM quiz_questions
ORDER BY created_at ASC;

-- name: GetQuizQuestion :one
SELECT id, slug, topic, prompt, choices, correct_index, explanation, created_at
FROM quiz_questions
WHERE id = $1 LIMIT 1;

-- name: CreateQuizQuestion :one
INSERT INTO quiz_questions (
    slug, topic, prompt, choices, correct_index, explanation
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, slug, topic, prompt, choices, correct_index, explanation, created_at;

-- name: GetQuizQuestionBySlug :one
SELECT id, slug, topic, prompt, choices, correct_index, explanation, created_at
FROM quiz_questions
WHERE slug = $1 LIMIT 1;

-- name: CreateQuizAttempt :one
INSERT INTO quiz_attempts (
    user_id, question_id, selected_index, is_correct
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, user_id, question_id, selected_index, is_correct, created_at;

-- name: ListQuizAttemptsByUser :many
SELECT id, user_id, question_id, selected_index, is_correct, created_at
FROM quiz_attempts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCorrectAttemptsByUser :one
SELECT COUNT(*)
FROM quiz_attempts
WHERE user_id = $1 AND is_correct = true;
