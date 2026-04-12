ALTER TABLE projects
ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE projects
SET updated_at = created_at
WHERE updated_at IS NULL;

ALTER TABLE projects
ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE projects
ALTER COLUMN updated_at SET DEFAULT now();
