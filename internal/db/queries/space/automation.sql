-- name: InsertAutomationRule :execresult
INSERT INTO automation_rules (name, description, trigger_type, priority, conditions, actions,
	is_active, stop_processing, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?);

-- name: UpdateAutomationRule :exec
UPDATE automation_rules SET name=?, description=?, trigger_type=?, priority=?,
	conditions=?, actions=?, is_active=?, stop_processing=?, updated_at=? WHERE id=?;

-- name: DeleteAutomationRule :exec
DELETE FROM automation_rules WHERE id = ?;

-- name: ToggleAutomationRule :exec
UPDATE automation_rules SET is_active=?, updated_at=? WHERE id=?;

-- name: SetAutomationPriority :exec
UPDATE automation_rules SET priority=? WHERE id=?;

-- name: ListAutomationLogs :many
SELECT id, rule_id, trigger_entity_type, trigger_entity_id, actions_executed, status, error_message, created_at
FROM automation_rule_logs WHERE rule_id = ? ORDER BY created_at DESC LIMIT 50;

-- name: IncrementAutomationRuns :exec
UPDATE automation_rules SET runs_count = runs_count + 1, last_run_at=?, updated_at=? WHERE id=?;

-- name: InsertAutomationLog :exec
INSERT INTO automation_rule_logs (rule_id, trigger_entity_type, trigger_entity_id, actions_executed, status, error_message, created_at)
VALUES (?,?,?,?,?,?,?);

-- name: ListAutomationRules :many
SELECT id, name, description, trigger_type, priority, conditions, actions,
	is_active, stop_processing, runs_count, last_run_at, created_at, updated_at
FROM automation_rules
WHERE id = COALESCE(sqlc.narg('id'), id)
  AND is_active = COALESCE(sqlc.narg('active_only'), is_active)
  AND trigger_type = COALESCE(sqlc.narg('trigger_type'), trigger_type)
ORDER BY priority, id;
