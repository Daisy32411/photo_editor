-- +migrate Up
ALTER TABLE tasks
ADD COLUMN user_id UUID REFERENCES users(id);

CREATE INDEX idx_tasks_user_id ON tasks(user_id);

-- +migrate Down
DROP INDEX idx_tasks_user_id;

ALTER TABLE tasks
DROP COLUMN user_id;