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
    address VARCHAR(100),
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

-- TODO: add unique index on `contests.wallet_id`
-- TODO: encrypt the mnemonic before saving
CREATE TABLE wallets (
    id SERIAL PRIMARY KEY,
    address VARCHAR(100) NOT NULL,
    mnemonic TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TYPE award_type AS ENUM ('no', 'pool', 'sponsored');

CREATE TABLE contests (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(64) NOT NULL,
    description VARCHAR(300) DEFAULT '' NOT NULL,
    award_type award_type NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration_mins INTEGER NOT NULL CHECK (duration_mins >= 0),
    max_entries INTEGER DEFAULT 0 NOT NULL CHECK (max_entries >= 0),
    allow_late_join BOOLEAN DEFAULT true NOT NULL,
    wallet_id INTEGER REFERENCES wallets(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TYPE difficulty AS ENUM ('easy', 'mid', 'hard');

CREATE TABLE problems (
    id SERIAL PRIMARY KEY,
    writer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(64) NOT NULL,
    statement TEXT NOT NULL,
    difficulty difficulty NOT NULL,
    time_limit_ms INTEGER DEFAULT 2000 NOT NULL CHECK (time_limit_ms >= 0),
    memory_limit_mb INTEGER DEFAULT 128 NOT NULL CHECK (memory_limit_mb >= 0),
    checker VARCHAR(10) NOT NULL DEFAULT 'tokens',
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE test_cases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    is_example BOOLEAN DEFAULT false NOT NULL,
    UNIQUE (problem_id, ordinal)
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
    is_paid BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    UNIQUE (contest_id, user_id)
);

CREATE TABLE submissions (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    verdict VARCHAR(30) NOT NULL DEFAULT 'not_judged',
    code TEXT NOT NULL,
    language VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE testing_reports (
    id SERIAL PRIMARY KEY,
    submission_id INTEGER NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    passed_tests_count INTEGER DEFAULT 0 NOT NULL CHECK (passed_tests_count >= 0),
    total_tests_count INTEGER DEFAULT 0 NOT NULL CHECK (total_tests_count >= 0),
    first_failed_test_id INTEGER REFERENCES test_cases(id),
    first_failed_test_output TEXT DEFAULT '',
    stderr TEXT DEFAULT '' NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
