-- name: CreateFlashcard :one
INSERT INTO flashcards (
    user_id, question_id, front, back
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, user_id, question_id, front, back, is_known, created_at, updated_at;

-- name: GetFlashcard :one
SELECT id, user_id, question_id, front, back, is_known, created_at, updated_at
FROM flashcards
WHERE id = $1 LIMIT 1;

-- name: ListFlashcardsByUser :many
SELECT id, user_id, question_id, front, back, is_known, created_at, updated_at
FROM flashcards
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: SetFlashcardKnown :one
UPDATE flashcards
SET is_known = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, question_id, front, back, is_known, created_at, updated_at;

-- name: DeleteFlashcard :exec
DELETE FROM flashcards
WHERE id = $1 AND user_id = $2;
