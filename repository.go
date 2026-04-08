package main

import (
	"database/sql"
	"fmt"
)

type TodoRepo struct {
	db *sql.DB
}

func NewTodoRepo(db *sql.DB) *TodoRepo {
	return &TodoRepo{db: db}
}

func (r *TodoRepo) Create(task string) (Todo, error) {
	insertSQL := `INSERT INTO todos (task) VALUES (?)`
	result, err := r.db.Exec(insertSQL, task)
	if err != nil {
		return Todo{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Todo{}, err
	}
	var todo Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}
	return todo, nil
}

func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error) {
	var todos []Todo
	offset := (page - 1) * LIMIT
	queryAllSQL := `SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`
	rows, err := r.db.Query(queryAllSQL, LIMIT, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt); err != nil {
			return nil, 0, err
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = r.db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return todos, total, nil
}

func (r *TodoRepo) Delete(id int) (int64, error) {
	deleteSQL := `DELETE FROM todos WHERE id = ?`
	result, err := r.db.Exec(deleteSQL, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *TodoRepo) Update(id int, task string) (Todo, error) {
	updateSQL := `UPDATE todos SET task = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.Exec(updateSQL, task, id)
	if err != nil {
		return Todo{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Todo{}, err
	}
	if rowsAffected == 0 {
		return Todo{}, fmt.Errorf("todo with ID %d not found", id)
	}
	var todo Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}
	return todo, nil
}

func (r *TodoRepo) Toggle(id int) (Todo, error) {
	updateSQL := `UPDATE todos SET done = NOT done, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.Exec(updateSQL, id)
	if err != nil {
		return Todo{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Todo{}, err
	}
	if rowsAffected == 0 {
		return Todo{}, fmt.Errorf("todo with ID %d not found", id)
	}
	var todo Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}
	return todo, nil
}
