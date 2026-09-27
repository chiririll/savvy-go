ALTER TABLE categories ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0;

-- Prefer promoting the existing "Other" category as default per type.
UPDATE categories SET is_default = 1 WHERE name = '#OTHER' AND type = 'expense';
UPDATE categories SET is_default = 1 WHERE name = '#OTHER_INCOME' AND type = 'income';

-- Ensure every type that already has categories but no default gets one
-- (oldest category wins) even if #OTHER was renamed or removed.
UPDATE categories SET is_default = 1
WHERE type = 'expense'
  AND id = (SELECT MIN(id) FROM categories WHERE type = 'expense')
  AND NOT EXISTS (SELECT 1 FROM categories WHERE type = 'expense' AND is_default = 1);

UPDATE categories SET is_default = 1
WHERE type = 'income'
  AND id = (SELECT MIN(id) FROM categories WHERE type = 'income')
  AND NOT EXISTS (SELECT 1 FROM categories WHERE type = 'income' AND is_default = 1);
