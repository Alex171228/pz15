-- scripts/explain.sql
\echo '--- OFFSET pagination (плохо на больших OFFSET) ---'
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, title, content, created_at
FROM notes
ORDER BY created_at DESC, id DESC
OFFSET 20000 LIMIT 20;

\echo '--- Keyset pagination (хорошо) ---'
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, title, content, created_at
FROM notes
WHERE (created_at, id) < (now(), 9223372036854775807)
ORDER BY created_at DESC, id DESC
LIMIT 20;

\echo '--- Search by title (GIN index) ---'
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, title
FROM notes
WHERE to_tsvector('simple', title) @@ plainto_tsquery('simple', 'title');
