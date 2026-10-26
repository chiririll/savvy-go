-- Manual order of transactions within one day. NULL means "not placed by
-- hand": the transaction sorts by its id, i.e. by creation. A placed one holds
-- a slot taken from its day's own sort keys COALESCE(day_order, id), so new
-- transactions (larger ids) still land after it.
ALTER TABLE transactions ADD COLUMN day_order INTEGER;
