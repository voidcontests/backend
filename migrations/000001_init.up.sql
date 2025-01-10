CREATE TABLE contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(64) NOT NULL,
    description VARCHAR(512) DEFAULT '' NOT NULL,
    creator_address VARCHAR(70) NOT NULL,
    starting_at TIMESTAMP NOT NULL,
    duration_mins INTEGER NOT NULL,
    is_draft BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE problems
(
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(id),
    title VARCHAR(64) NOT NULL,
    statement TEXT DEFAULT '' NOT NULL,
    difficulty VARCHAR(10) NOT NULL,
    writer_address VARCHAR(70) NOT NULL,
    input TEXT NOT NULL,
    answer TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
