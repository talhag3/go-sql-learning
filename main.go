package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "todos.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        task TEXT NOT NULL,
        done BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
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

func main() {
	args := os.Args[1:]
	db := InitDB()
	defer CloseDB(db)

	if len(args) == 0 {
		fmt.Println("No command provided")
		PrintHelp()
		return
	}

	repo := NewTodoRepo(db)

	switch args[0] {
	case "help", "h":
		PrintHelp()
	case "list":
		handleList(repo, args)
	case "create":
		handleCreate(repo, args)
	case "delete":
		handleDelete(repo, args)
	case "update":
		handleUpdate(repo, args)
	case "toggle":
		handleToggle(repo, args)
	case "version", "v":
		fmt.Println("App Version: 1.0.0")
	default:
		fmt.Println("Unknown command")
		PrintHelp()
	}
}
