package db

// SQLite has no LATERAL joins. Group each ledger once and join by task rather
// than issuing separate queries per task or repeatedly scanning its history.
const sqliteTaskListMetricsSQL = `
WITH token_metrics AS (
 SELECT exploration_id,sum(input_tokens) input_tokens,sum(output_tokens) output_tokens,
 sum(cache_read_tokens) cache_read_tokens,sum(cache_write_tokens) cache_write_tokens
 FROM activity WHERE kind='result' GROUP BY exploration_id
), latest_activity AS (
 SELECT exploration_id,CAST(sw_epoch(max(created_at)) AS INTEGER) created_at
 FROM activity GROUP BY exploration_id
), goal_metrics AS (
 SELECT exploration_id,count(*) total,count(*) FILTER(WHERE state='met') met
 FROM exploration_nodes WHERE kind='goal' GROUP BY exploration_id
), intent_metrics AS (
 SELECT exploration_id,count(*) running FROM exploration_nodes
 WHERE kind='intent' AND state='running' GROUP BY exploration_id
), finding_metrics AS (
 SELECT task_id,count(*) FILTER(WHERE severity='critical') critical,
 count(*) FILTER(WHERE severity='high') high,count(*) FILTER(WHERE severity='medium') medium,
 count(*) FILTER(WHERE severity='low') low FROM findings GROUP BY task_id
)
SELECT task.exploration_id,
 COALESCE(token_metrics.input_tokens,0),COALESCE(token_metrics.output_tokens,0),
 COALESCE(token_metrics.cache_read_tokens,0),COALESCE(token_metrics.cache_write_tokens,0),
 COALESCE(latest_activity.created_at,0),COALESCE(goal_metrics.total,0),COALESCE(goal_metrics.met,0),
 COALESCE(intent_metrics.running,0),COALESCE(finding_metrics.critical,0),COALESCE(finding_metrics.high,0),
 COALESCE(finding_metrics.medium,0),COALESCE(finding_metrics.low,0)
FROM tasks task
LEFT JOIN token_metrics ON token_metrics.exploration_id=task.exploration_id
LEFT JOIN latest_activity ON latest_activity.exploration_id=task.exploration_id
LEFT JOIN goal_metrics ON goal_metrics.exploration_id=task.exploration_id
LEFT JOIN intent_metrics ON intent_metrics.exploration_id=task.exploration_id
LEFT JOIN finding_metrics ON finding_metrics.task_id=task.id
WHERE task.deleted_at IS NULL`
