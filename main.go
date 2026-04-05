package main

import (
	"database/sql"
	"fmt"
	"log"

	// This is a blank import - only needed for side effects
	// It registers the sqlite3 driver with database/sql
	_ "github.com/mattn/go-sqlite3"
)

// ============================================
// WHAT IS database/sql?
// ============================================
// database/sql is Go's standard library package for
// working with SQL databases. It provides:
//
// 1. A generic interface - works with any SQL database
// 2. Connection pooling - manages connections efficiently
// 3. Prepared statements - prevents SQL injection
// 4. Transaction support - ensures data consistency
//
// The actual database-specific code is in "drivers"
// that you import with _ (blank import)
// ============================================

func main() {
	// ============================================
	// STEP 1: Open a database connection
	// ============================================
	// sql.Open doesn't actually connect to the database!
	// It just validates the driver name and prepares the connection string.
	// The actual connection happens when you execute a query.
	//
	// Parameters:
	// - "sqlite3": The driver name (registered by the blank import)
	// - "todos.db": Connection string (for SQLite, it's the file path)
	//   If file doesn't exist, SQLite creates it

	db, err := sql.Open("sqlite3", "todos.db")
	if err != nil {
		// log.Fatal calls log.Print then os.Exit(1)
		log.Fatal("Failed to open database:", err)
	}

	// CRITICAL: Always close the database when done
	// defer ensures this runs when main() returns
	// Even if there's a panic, defer still runs
	defer db.Close()

	// ============================================
	// STEP 2: Verify the connection actually works
	// ============================================
	// Ping actually connects to the database and checks if it's alive
	// This is important because sql.Open is lazy

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("✅ Successfully connected to SQLite database!")

	// ============================================
	// STEP 3: Create a table
	// ============================================
	// Exec is used for SQL statements that don't return rows:
	// - CREATE TABLE
	// - INSERT
	// - UPDATE
	// - DELETE
	//
	// It returns:
	// - Result: Contains info about what was affected
	// - Error: If something went wrong

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        -- INTEGER PRIMARY KEY + AUTOINCREMENT = auto-incrementing integer ID
        -- In SQLite, this is the standard way to create auto-increment IDs
        
        title TEXT NOT NULL,
        -- TEXT = string type
        -- NOT NULL = this field cannot be empty
        
        description TEXT,
        -- No NOT NULL = this field can be NULL (optional)
        
        completed BOOLEAN DEFAULT FALSE,
        -- BOOLEAN = true/false (SQLite stores as 0 or 1)
        -- DEFAULT FALSE = if not specified, defaults to false
        
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        -- TIMESTAMP = date/time
        -- DEFAULT CURRENT_TIMESTAMP = automatically set to current time
        
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    `

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	fmt.Println("✅ Table 'todos' created (or already exists)")

	// ============================================
	// STEP 4: Insert a record
	// ============================================

	insertSQL := `
    INSERT INTO todos (title, description) 
    VALUES (?, ?)
    `
	// ? are placeholders (parameterized queries)
	// They prevent SQL injection - the driver safely escapes the values

	// Exec returns a Result which has:
	// - LastInsertId(): The auto-generated ID
	// - RowsAffected(): Number of rows affected

	result, err := db.Exec(insertSQL, "Learn Go", "Study Go fundamentals")
	if err != nil {
		log.Fatal("Failed to insert todo:", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		log.Fatal("Failed to get last insert ID:", err)
	}

	fmt.Printf("✅ Inserted todo with ID: %d\n", id)

	// ============================================
	// STEP 5: Insert multiple records
	// ============================================

	todos := []struct {
		Title       string
		Description string
	}{
		{"Learn PostgreSQL", "Study PostgreSQL and pgx"},
		{"Build CLI App", "Create a todo CLI application"},
		{"Learn SQLC", "Generate Go code from SQL"},
	}

	for _, todo := range todos {
		result, err := db.Exec(
			insertSQL,
			todo.Title,
			todo.Description,
		)
		if err != nil {
			log.Printf("Failed to insert '%s': %v", todo.Title, err)
			continue
		}
		id, _ := result.LastInsertId()
		fmt.Printf("✅ Inserted todo with ID: %d\n", id)
	}

	// ============================================
	// STEP 6: Query a single row
	// ============================================
	// QueryRow is used when you expect exactly ONE row
	// If no rows match, it returns sql.ErrNoRows
	// If multiple rows match, it only returns the first

	var (
		title       string
		description string
		completed   bool
	)

	querySingleSQL := `
    SELECT title, description, completed 
    FROM todos 
    WHERE id = ?
    `

	// QueryRow returns a *Row
	// You must call Scan() to actually execute the query and read values
	err = db.QueryRow(querySingleSQL, 1).Scan(
		&title, // & means "address of" - Scan writes to this variable
		&description,
		&completed,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("⚠️ No todo found with ID 1")
		} else {
			log.Fatal("Failed to query todo:", err)
		}
	} else {
		fmt.Printf("📋 Found todo: %s - %s (completed: %v)\n",
			title, description, completed)
	}

	// ============================================
	// STEP 7: Query multiple rows
	// ============================================
	// Query is used when you expect ZERO or MORE rows
	// It returns *Rows which you iterate over

	queryAllSQL := `SELECT id, title, description, completed FROM todos`

	rows, err := db.Query(queryAllSQL)
	if err != nil {
		log.Fatal("Failed to query todos:", err)
	}

	// CRITICAL: Always close rows when done
	// This releases the database connection back to the pool
	defer rows.Close()

	fmt.Println("\n📋 All Todos:")
	fmt.Println("─────────────────────────────────────────")

	// Next() moves to the next row
	// Returns false when there are no more rows
	for rows.Next() {
		var (
			id          int
			title       string
			description string
			completed   bool
		)

		// Scan reads column values into variables
		// Order must match SELECT order
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

	// CRITICAL: Always check for errors after the loop
	// Errors can occur during iteration
	err = rows.Err()
	if err != nil {
		log.Fatal("Error iterating rows:", err)
	}

	// ============================================
	// STEP 8: Update a record
	// ============================================

	updateSQL := `
    UPDATE todos 
    SET completed = ?, updated_at = CURRENT_TIMESTAMP 
    WHERE id = ?
    `

	result, err = db.Exec(updateSQL, true, 1)
	if err != nil {
		log.Fatal("Failed to update todo:", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}

	fmt.Printf("\n✅ Updated %d row(s)\n", rowsAffected)

	// ============================================
	// STEP 9: Delete a record
	// ============================================

	deleteSQL := `DELETE FROM todos WHERE id = ?`

	result, err = db.Exec(deleteSQL, 4) // Delete the last one
	if err != nil {
		log.Fatal("Failed to delete todo:", err)
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}

	fmt.Printf("✅ Deleted %d row(s)\n", rowsAffected)

	// ============================================
	// STEP 10: Demonstrate prepared statements
	// ============================================
	// Prepared statements are pre-compiled SQL
	// Benefits:
	// 1. Better performance for repeated queries
	// 2. Automatic SQL injection prevention
	// 3. Type safety

	fmt.Println("\n📝 Using Prepared Statements:")

	// Prepare the statement once
	stmt, err := db.Prepare(`
        INSERT INTO todos (title, description) 
        VALUES (?, ?)
    `)
	if err != nil {
		log.Fatal("Failed to prepare statement:", err)
	}
	defer stmt.Close() // Close when done

	// Execute multiple times with different values
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

	// Print final state
	fmt.Println("\n📋 Final Todo List:")
	printAllTodos(db)
}

// Helper function to print all todos
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
