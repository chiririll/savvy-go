-- Manual order of transactions within one day. NULL means "not placed by
-- hand": the transaction sorts by its id, i.e. by creation. A placed one holds
-- a slot taken from its day's own sort keys COALESCE(day_order, id), so new
-- transactions (larger ids) still land after it.
ALTER TABLE transactions ADD COLUMN day_order INTEGER;

-- Legs of one payment. A payment made with several accounts or currencies
-- (cash and card, or euros with change in dinars) is several ordinary
-- transactions that share a group_uuid; the group is these rows and nothing
-- else, so every balance and report works on the legs as before. Rows outside
-- a group have NULL.
ALTER TABLE transactions ADD COLUMN group_uuid TEXT;
CREATE INDEX IF NOT EXISTS transactions_group_idx ON transactions (group_uuid) WHERE group_uuid IS NOT NULL;
