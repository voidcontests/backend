CREATE TABLE contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(64) NOT NULL,
    description VARCHAR(512) DEFAULT '' NOT NULL,
    problem_ids []INTEGER DEFAULT '{}',
    creator_address VARCHAR(64) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    duration INTERVAL NOT NULL,
    max_slots INTEGER NOT NULL,
    -- applied_participants INTEGER DEFAULT 0 NOT NULL,
    is_draft BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE problems
(
    id SERIAL PRIMARY KEY,
    title VARCHAR(64) NOT NULL,
    task TEXT DEFAULT '' NOT NULL,
    writer_address VARCHAR(64) NOT NULL,
    -- kind ? -- TODO: maybe create an enum for problem kind like (single-answer, multi-answer, code)
    -- difficulty ???
    input TEXT NOT NULL,
    answer TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
