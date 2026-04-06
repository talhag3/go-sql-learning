package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID        int
	Task      string
	Done      bool
	CreatedAt time.Time
}

const LIMIT = 10

/* DB Management */

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "todos.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// fmt.Println("Successfully connected to SQLite database!")

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        task TEXT NOT NULL,
        done BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	fmt.Println("Table 'todos' created (or already exists)")

	return db
}

func CloseDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		log.Fatal("Error closing the db", err)
	}

	fmt.Println("Db connection closed")
}

func createTask(db *sql.DB, task string) (Todo, error) {
	insertSQL := `INSERT INTO todos (task) VALUES (?)`

	result, err := db.Exec(insertSQL, task)
	if err != nil {
		return Todo{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Todo{}, err
	}

	// Query the inserted record
	var todo Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}

	return todo, nil
}

func getTodos(db *sql.DB, page int) ([]Todo, int, error) {
	var todos []Todo
	offset := (page - 1) * LIMIT
	queryAllSQL := `SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`

	rows, err := db.Query(queryAllSQL, LIMIT, offset)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		todos = append(todos, todo)
	}

	err = rows.Err()
	if err != nil {
		return nil, 0, err
	}

	/* Get total records count */

	countSql := "SELECT  COUNT(*) from todos"
	var total int
	err = db.QueryRow(countSql).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return todos, total, nil
}

func deleteTask(db *sql.DB, id int) (int64, error) {
	deleteSQL := `DELETE FROM todos WHERE id = ?`

	result, err := db.Exec(deleteSQL, id)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func updateTask(db *sql.DB, id int, task string) (Todo, error) {
	updateSQL := `UPDATE todos SET task = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := db.Exec(updateSQL, task, id)
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
	err = db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}

	return todo, nil
}

func toggleTask(db *sql.DB, id int) (Todo, error) {
	updateSQL := `UPDATE todos SET done = NOT done, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err := db.Exec(updateSQL, id)
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
	err = db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return Todo{}, err
	}

	return todo, nil
}

/* ------------------------- */

func PrintHelp() {
	fmt.Println("\n Available Commands:")
	fmt.Println("  help | h              Show help")
	fmt.Println("  list <page>           List task (default page = 1)")
	fmt.Println("  create <task>         Create task")
	fmt.Println("  delete <id>           Delete task by ID")
	fmt.Println("  update <id> <task>    Update task")
	fmt.Println("  toggle  <id>			 Toggle task")
	fmt.Println("  version | v           Show version")
}

func handleList(db *sql.DB, args []string) {
	page := 1

	if len(args) > 1 {
		p, err := strconv.Atoi(args[1])
		if err != nil || p <= 0 {
			fmt.Println("Invalid page number")
			return
		}
		page = p
	}

	if len(args) > 2 {
		fmt.Println("Too many arguments for 'list'")
		return
	}

	todos, total, err := getTodos(db, page)

	if err != nil {
		fmt.Println("Error getting the todos", err)
		return
	}

	totalPages := (total + LIMIT - 1) / LIMIT
	fmt.Printf("\n=== Todos (Page %d of %d) ===\n", page, totalPages)
	fmt.Print("\n─────────────────────────────────────────\n\n")

	for _, todo := range todos {
		status := "❌"
		if todo.Done {
			status = "✅"
		}
		fmt.Printf("%s [%d] %s\n", status, todo.ID, todo.Task)
	}

	fmt.Print("\n\n─────────────────────────────────────────\n")
	fmt.Printf("Total: %d todos\n", total)
}

func handleCreate(db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Println("task text is required")
		fmt.Println("Usage: create <task>")
		return
	}

	task := strings.Join(args[1:], " ")

	todo, err := createTask(db, task)

	if err != nil {
		fmt.Println("Error creating task:", err)
		return
	}

	fmt.Println("\nTodo Created Successfully!")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("  ID:        %d\n", todo.ID)
	fmt.Printf("  Task:      %s\n", todo.Task)
	fmt.Printf("  Status:    %s\n", map[bool]string{true: "✅ Done", false: "❌ Pending"}[todo.Done])
	fmt.Printf("  Created:   %s\n", todo.CreatedAt.Format("2006-01-02 15:04:05"))
}

func handleDelete(db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Println("ID is required")
		fmt.Println("Usage: delete <id>")
		return
	}

	id, err := strconv.Atoi(args[1])
	if err != nil || id <= 0 {
		fmt.Println("Invalid ID")
		return
	}

	rowsAffected, err := deleteTask(db, id)
	if err != nil {
		fmt.Println("Error deleting task:", err)
		return
	}

	if rowsAffected == 0 {
		fmt.Printf("⚠️  No todo found with ID: %d\n", id)
		return
	}

	fmt.Printf("✅ Deleted todo #%d successfully\n", id)
}

func handleUpdate(db *sql.DB, args []string) {
	if len(args) < 3 {
		fmt.Println("ID and task are required")
		fmt.Println("Usage: update <id> <task>")
		return
	}

	id, err := strconv.Atoi(args[1])
	if err != nil || id <= 0 {
		fmt.Println("Invalid ID")
		return
	}

	task := strings.Join(args[2:], " ")

	todo, err := updateTask(db, id, task)
	if err != nil {
		fmt.Println("Error updating task:", err)
		return
	}

	fmt.Println("\n✅ Todo Updated Successfully!")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("  ID:        %d\n", todo.ID)
	fmt.Printf("  Task:      %s\n", todo.Task)
	fmt.Printf("  Status:    %s\n", map[bool]string{true: "✅ Done", false: "❌ Pending"}[todo.Done])
	fmt.Printf("  Created:   %s\n", todo.CreatedAt.Format("2006-01-02 15:04:05"))
}

func handleToggle(db *sql.DB, args []string) {
	if len(args) < 2 {
		fmt.Println("ID is required")
		fmt.Println("Usage: toggle <id>")
		return
	}

	id, err := strconv.Atoi(args[1])
	if err != nil || id <= 0 {
		fmt.Println("Invalid ID")
		return
	}

	todo, err := toggleTask(db, id)
	if err != nil {
		fmt.Println("Error toggling task:", err)
		return
	}

	status := "❌ Pending"
	if todo.Done {
		status = "✅ Done"
	}

	fmt.Println("\n✅ Todo Toggled Successfully!")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("  ID:        %d\n", todo.ID)
	fmt.Printf("  Task:      %s\n", todo.Task)
	fmt.Printf("  Status:    %s\n", status)
}

func main() {
	args := os.Args[1:]
	db := InitDB()
	defer CloseDB(db)

	if len(args) == 0 {
		fmt.Println("No command provided")
		PrintHelp()
		return
	}

	switch args[0] {

	case "help", "h":
		PrintHelp()

	case "list":
		handleList(db, args)

	case "create":
		handleCreate(db, args)

	case "delete":
		handleDelete(db, args)

	case "update":
		handleUpdate(db, args)

	case "toggle":
		handleToggle(db, args)

	case "version", "v":
		fmt.Println("App Version: 1.0.0")

	default:
		fmt.Println("Unknown command")
		PrintHelp()
	}

}
