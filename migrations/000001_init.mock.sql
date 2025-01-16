INSERT INTO users (address, username) VALUES ('0:ba46ec7afab69d343db0740de78cd9bea92901f8acd7287d126c04d98109a8ea', 'ndbtea');

INSERT INTO contests (creator_id, title, description, starting_at, duration_mins, is_draft) VALUES (1, 'The Void Round 0', 'dev/.', '2025-02-04 17:00:00', 180, false);

INSERT INTO problems (contest_id, writer_id, title, statement, difficulty, input, answer) VALUES (1, 1, 'Toilet Solvings', 'Sum numbers from input', 'insane', '38\n31', '69');

INSERT INTO entries (contest_id, user_id) VALUES (1, 1);
