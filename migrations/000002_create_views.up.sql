CREATE VIEW leaderboard AS
SELECT
    e.contest_id,
    u.id AS user_id,
    u.username,
    COALESCE(SUM(
        CASE
            WHEN p.difficulty = 'easy' THEN 1
            WHEN p.difficulty = 'mid' THEN 3
            WHEN p.difficulty = 'hard' THEN 5
            ELSE 0
        END
    ), 0) AS points
FROM users u
JOIN entries e ON u.id = e.user_id
LEFT JOIN (
    SELECT DISTINCT entry_id, problem_id
    FROM submissions
    WHERE verdict = 'ok'
) s ON e.id = s.entry_id
LEFT JOIN problems p ON s.problem_id = p.id
GROUP BY e.contest_id, u.id, u.username;


CREATE VIEW contest_details AS
SELECT
    c.id,
    c.creator_id,
    u.username AS creator_username,
    c.title,
    c.description,
    c.award_type,
    c.start_time,
    c.end_time,
    c.duration_mins,
    c.max_entries,
    c.allow_late_join,
    c.wallet_id,
    COUNT(e.id) AS participants_count,
    c.created_at
FROM contests c
JOIN users u ON u.id = c.creator_id
LEFT JOIN entries e ON e.contest_id = c.id
GROUP BY c.id, u.username;


CREATE VIEW problem_details AS
SELECT
    p.id,
    p.writer_id,
    u.username AS writer_username,
    p.title,
    p.statement,
    p.difficulty,
    p.time_limit_ms,
    p.memory_limit_mb,
    p.checker,
    p.created_at
FROM problems p
JOIN users u ON u.id = p.writer_id;


CREATE VIEW problem_statuses AS
SELECT
    s.problem_id,
    s.entry_id,
    CASE
        WHEN COUNT(*) FILTER (WHERE s.verdict = 'ok') > 0 THEN 'accepted'
        WHEN COUNT(*) > 0 THEN 'tried'
    END AS status
FROM submissions s
GROUP BY s.entry_id, s.problem_id;


CREATE VIEW contest_problemsets AS
SELECT
    p.id AS problem_id,
    cp.charcode,
    cp.contest_id,
    p.writer_id,
    u.username AS writer_username,
    p.title,
    p.statement,
    p.difficulty,
    p.time_limit_ms,
    p.memory_limit_mb,
    p.checker,
    p.created_at
FROM problems p
JOIN contest_problems cp ON p.id = cp.problem_id
JOIN users u ON u.id = p.writer_id;


CREATE VIEW submission_details AS
SELECT
    s.id,
    s.entry_id,
    e.contest_id,
    s.problem_id,
    e.user_id,
    u.username,
    s.status,
    s.verdict,
    s.code,
    s.language,
    s.created_at
FROM submissions s
JOIN entries e ON s.entry_id = e.id
JOIN users u ON e.user_id = u.id;
