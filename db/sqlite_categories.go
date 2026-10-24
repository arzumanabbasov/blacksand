package db

import "database/sql"

func (d *DB) createSQLiteTaskCategory(name, nkey string) (*TaskCategory, error) {
	category, err := scanTaskCategory(d.QueryRow(`INSERT INTO task_categories(name,nkey) VALUES($1,$2) ON CONFLICT(nkey) DO NOTHING RETURNING id,name,nkey,0,created_at,updated_at`, name, nkey))
	if err == sql.ErrNoRows {
		return nil, ErrTaskCategoryNameConflict
	}
	if err != nil {
		return nil, err
	}
	return &category, nil
}
func (d *DB) renameSQLiteTaskCategory(id int64, name, nkey string) (*TaskCategory, error) {
	category, err := scanTaskCategory(d.QueryRow(`UPDATE task_categories SET name=$2,nkey=$3 WHERE id=$1 RETURNING id,name,nkey,(SELECT count(*) FROM tasks WHERE category_id=$1 AND deleted_at IS NULL),created_at,updated_at`, id, name, nkey))
	if err == sql.ErrNoRows {
		return nil, ErrTaskCategoryNotFound
	}
	if taskCategoryUniqueViolation(err) {
		return nil, ErrTaskCategoryNameConflict
	}
	if err != nil {
		return nil, err
	}
	return &category, nil
}
func (d *DB) setSQLiteTaskCategory(taskID, categoryID int64) (*TaskCategory, error) {
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM tasks WHERE id=$1 AND deleted_at IS NULL)`, taskID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTaskCategoryTaskNotFound
	}
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM task_categories WHERE id=$1)`, categoryID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTaskCategoryNotFound
	}
	if _, err := tx.Exec(`UPDATE tasks SET category_id=$2 WHERE id=$1 AND deleted_at IS NULL`, taskID, categoryID); err != nil {
		return nil, err
	}
	category, err := scanTaskCategory(tx.QueryRow(`SELECT id,name,nkey,(SELECT count(*) FROM tasks WHERE category_id=$1 AND deleted_at IS NULL),created_at,updated_at FROM task_categories WHERE id=$1`, categoryID))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &category, nil
}
