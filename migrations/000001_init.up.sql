CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(20) UNIQUE NOT NULL,
    created_problems_limit INTEGER NOT NULL,
    created_contests_limit INTEGER NOT NULL,
    is_default BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

INSERT INTO roles (name, created_problems_limit, created_contests_limit, is_default) VALUES
    ('admin', -1, -1, false),
    ('unlimited', -1, -1, false),
    ('limited', 10, 2, true),
    ('banned', 0, 0, false);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TYPE problem_kind AS ENUM ('text_answer_problem', 'coding_problem');
CREATE TYPE verdict AS ENUM (
    'pending', 'running', 'ok', 'wrong_answer',
    'runtime_error', 'compilation_error', 'time_limit_exceeded'
);

CREATE TABLE contests (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(64) NOT NULL,
    description VARCHAR(300) DEFAULT '' NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration_mins INTEGER NOT NULL CHECK (duration_mins > 0),
    max_entries INTEGER DEFAULT 0 NOT NULL CHECK (max_entries >= 0),
    allow_late_join BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE problems (
    id SERIAL PRIMARY KEY,
    kind problem_kind NOT NULL,
    writer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(64) NOT NULL,
    statement TEXT DEFAULT '' NOT NULL,
    difficulty VARCHAR(10) NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    answer TEXT NOT NULL,
    time_limit_ms INTEGER DEFAULT 5000 NOT NULL CHECK (time_limit_ms > 0),
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE test_cases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_example BOOLEAN DEFAULT false NOT NULL
);

CREATE TABLE contest_problems (
    contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    charcode VARCHAR(2) NOT NULL,
    PRIMARY KEY (contest_id, problem_id),
    UNIQUE (contest_id, charcode)
);

CREATE TABLE entries (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    UNIQUE (contest_id, user_id)
);

CREATE TABLE submissions (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    verdict verdict NOT NULL,
    answer TEXT NOT NULL,
    code TEXT NOT NULL,
    language VARCHAR(20) NOT NULL,
    passed_tests_count INTEGER DEFAULT 0 NOT NULL CHECK (passed_tests_count >= 0),
    stderr TEXT NOT NULL,
    locked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE failed_tests (
    id SERIAL PRIMARY KEY,
    submission_id INTEGER NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    input TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    actual_output TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
