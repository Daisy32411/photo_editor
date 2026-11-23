-- +migrate Up

CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'in_progress',
    result TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_task_status
        CHECK (status IN ('in_progress', 'ready', 'failed'))
);

-- +migrate Down

DROP TABLE tasks;