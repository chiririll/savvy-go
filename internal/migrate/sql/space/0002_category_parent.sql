-- Groups a category under another of the same type, one level deep.
-- internal/ledger checks the type and depth; a parent's deletion frees its children.
ALTER TABLE categories ADD COLUMN parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL;