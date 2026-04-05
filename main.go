package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "todos.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	fmt.Println("✅ Successfully connected to SQLite database!")

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT,
        completed BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	fmt.Println("✅ Table 'todos' created (or already exists)")

	insertSQL := `INSERT INTO todos (title, description) VALUES (?, ?)`

	result, err := db.Exec(insertSQL, "Learn Go", "Study Go fundamentals")
	if err != nil {
		log.Fatal("Failed to insert todo:", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Fatal("Failed to get last insert ID:", err)
	}
	fmt.Printf("✅ Inserted todo with ID: %d\n", id)

	todos := []struct {
		Title       string
		Description string
	}{
		{"Learn PostgreSQL", "Study PostgreSQL and pgx"},
		{"Build CLI App", "Create a todo CLI application"},
		{"Learn SQLC", "Generate Go code from SQL"},
	}

	for _, todo := range todos {
		result, err := db.Exec(insertSQL, todo.Title, todo.Description)
		if err != nil {
			log.Printf("Failed to insert '%s': %v", todo.Title, err)
			continue
		}
		id, _ := result.LastInsertId()
		fmt.Printf("✅ Inserted todo with ID: %d\n", id)
	}

	var (
		title       string
		description string
		completed   bool
	)

	querySingleSQL := `SELECT title, description, completed FROM todos WHERE id = ?`

	err = db.QueryRow(querySingleSQL, 1).Scan(&title, &description, &completed)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("⚠️ No todo found with ID 1")
		} else {
			log.Fatal("Failed to query todo:", err)
		}
	} else {
		fmt.Printf("📋 Found todo: %s - %s (completed: %v)\n", title, description, completed)
	}

	queryAllSQL := `SELECT id, title, description, completed FROM todos`

	rows, err := db.Query(queryAllSQL)
	if err != nil {
		log.Fatal("Failed to query todos:", err)
	}
	defer rows.Close()

	fmt.Println("\n📋 All Todos:")
	fmt.Println("─────────────────────────────────────────")

	for rows.Next() {
		var (
			id          int
			title       string
			description string
			completed   bool
		)
		err := rows.Scan(&id, &title, &description, &completed)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		status := "❌"
		if completed {
			status = "✅"
		}
		fmt.Printf("%s [%d] %s - %s\n", status, id, title, description)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal("Error iterating rows:", err)
	}

	updateSQL := `UPDATE todos SET completed = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

	result, err = db.Exec(updateSQL, true, 1)
	if err != nil {
		log.Fatal("Failed to update todo:", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}
	fmt.Printf("\n✅ Updated %d row(s)\n", rowsAffected)

	deleteSQL := `DELETE FROM todos WHERE id = ?`

	result, err = db.Exec(deleteSQL, 4)
	if err != nil {
		log.Fatal("Failed to delete todo:", err)
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}
	fmt.Printf("✅ Deleted %d row(s)\n", rowsAffected)

	fmt.Println("\n📝 Using Prepared Statements:")

	stmt, err := db.Prepare(`INSERT INTO todos (title, description) VALUES (?, ?)`)
	if err != nil {
		log.Fatal("Failed to prepare statement:", err)
	}
	defer stmt.Close()

	prepTitles := []string{"Task A", "Task B", "Task C"}
	for _, t := range prepTitles {
		result, err := stmt.Exec(t, "Created with prepared statement")
		if err != nil {
			log.Printf("Failed to execute prepared statement: %v", err)
			continue
		}
		id, _ := result.LastInsertId()
		fmt.Printf("✅ Prepared statement inserted ID: %d\n", id)
	}

	fmt.Println("\n📋 Final Todo List:")
	printAllTodos(db)
}

func printAllTodos(db *sql.DB) {
	rows, err := db.Query(`
        SELECT id, title, completed 
        FROM todos 
        ORDER BY id
    `)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var title string
		var completed bool
		rows.Scan(&id, &title, &completed)

		status := "❌"
		if completed {
			status = "✅"
		}
		fmt.Printf("  %s [%d] %s\n", status, id, title)
	}
}
