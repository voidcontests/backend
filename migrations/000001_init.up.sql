CREATE TABLE roles
(
    id SERIAL PRIMARY KEY,
    name VARCHAR(20) UNIQUE NOT NULL,
    created_problems_limit INTEGER NOT NULL,
    created_contests_limit INTEGER NOT NULL,
    is_default BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

INSERT INTO
    roles (name, created_problems_limit, created_contests_limit)
VALUES
    ('admin', -1, -1),
    ('unlimited', -1, -1),
    ('limited', 10, 2),
    ('banned', 0, 0);

UPDATE roles SET is_default = true WHERE name = 'limited';

CREATE TABLE users
(
    id SERIAL PRIMARY KEY,
    address VARCHAR(70) UNIQUE NOT NULL,
    role_id INTEGER NOT NULL REFERENCES roles(id),
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE contests
(
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(64) NOT NULL,
    description VARCHAR(300) DEFAULT '' NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration_mins INTEGER NOT NULL,
    is_draft BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE problems
(
    id SERIAL PRIMARY KEY,
    writer_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(64) NOT NULL,
    statement TEXT DEFAULT '' NOT NULL,
    difficulty VARCHAR(10) NOT NULL,
    input TEXT NOT NULL,
    answer TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE contest_problems (
    contest_id INTEGER NOT NULL REFERENCES contests(id),
    problem_id INTEGER NOT NULL REFERENCES problems(id),
    charcode VARCHAR(2) NOT NULL,
    PRIMARY KEY (contest_id, problem_id)
);

CREATE TABLE entries
(
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TYPE verdict AS ENUM ('ok', 'wrong_answer');

CREATE TABLE submissions
(
    id SERIAL PRIMARY KEY,
    entry_id INTEGER NOT NULL REFERENCES entries(id),
    problem_id INTEGER NOT NULL REFERENCES problems(id),
    verdict verdict NOT NULL,
    answer TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
