ALTER TABLE tasks
ADD CONSTRAINT tasks_title_not_empty CHECK (title != ''),
ADD CONSTRAINT tasks_status_check CHECK (status in ('active', 'done'));