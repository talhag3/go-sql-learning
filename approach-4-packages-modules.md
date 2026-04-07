# Approach 4 — Multi-File & Multi-Package Architecture: Complete Guide

## The Big Picture

You currently have **one 400-line file** (`main.go`). Every real Go project splits code across **multiple files and packages**, each with a single responsibility. This guide teaches you how to do that — and you'll learn core Go concepts along the way.

```
WHERE YOU ARE NOW:                    WHERE YOU'RE GOING:
┌──────────────────────┐              ┌──────────────────────────────┐
│   main.go (402 lines)│              │  go-sql-learning/            │
│                      │              │  ├── go.mod                  │
│  • Todo struct       │              │  ├── main.go (~50 lines)     │
│  • InitDB / CloseDB  │    ──→       │  ├── model/                  │
│  • CRUD functions    │              │  │   └── todo.go             │
│  • handle* functions │              │  ├── repository/             │
│  • PrintHelp         │              │  │   ├── repo.go             │
│  • main()            │              │  │   └── crud.go             │
│                      │              │  └── handler/                │
└──────────────────────┘              │      ├── handler.go          │
                                      │      ├── handler_update.go   │
                                      │      └── help.go             │
                                      └──────────────────────────────┘
```

## What You'll Learn

| Concept | Why It Matters |
|---------|---------------|
| **Go Modules** (`go.mod`) | How Go identifies your project & manages dependencies |
| **Packages** | How to group related code; the unit of reuse in Go |
| **Visibility** (`Todo` vs `todo`) | Uppercase = public, lowercase = private — #1 beginner trap |
| **Imports** | How to use your own packages + stdlib + third-party |
| **Project Layout** | How real Go projects are organized |

---

## Prerequisite Concepts (Learn These First)

### 1. What is a Go Module?

A **module** is a collection of Go packages released together. It has:
- A **module path** (unique identifier, usually a URL)
- A **`go.mod`** file at the project root
- A **Go version**

```bash
# Check if you already have one
ls go.mod

# If not, create one
go mod init github.com/YOURNAME/go-sql-learning
```

Your `go.mod` looks like this:
```go
module github.com/YOURNAME/go-sql-learning

go 1.21

require github.com/mattn/go-sqlite3 v1.14.22
```

**The module path becomes your import prefix.** Every package inside your project is imported as `module-path/package-directory`.

### 2. What is a Package?

A **package** = a directory of `.go` files that all say `package <name>`.

```go
// model/todo.go
package model    // ← directory name = package name (STRONG convention)

type Todo struct { ... }
```

**Key facts:**
- Package name = directory name (by convention)
- One package can span multiple files in the same directory
- A package is the **smallest unit of visibility** — things are either visible within a package or outside it
- `package main` is special — it's where `func main()` lives and produces an executable

### 3. Visibility Rules (MEMORIZE THIS)

This is the **#1 thing** that breaks when moving from single-file to multi-package:

```go
package model

type Todo struct { ... }     // ← EXPORTED: starts with uppercase → usable from ANY package
type todo struct { ... }    // ← UNEXPORTED: starts with lowercase → ONLY usable inside "model"

func New() Todo { ... }     // ← EXPORTED
func helper() { ... }       // ← UNEXPORTED
```

| Name | Visible In | Example Usage |
|------|-----------|---------------|
| `Todo` (uppercase) | Any package that imports `model` | `var t model.Todo` ✅ |
| `todo` (lowercase) | Only inside `model` package | `model.todo{}` ❌ won't compile |
| `GetAll()` (uppercase) | Any importing package | `repo.GetAll()` ✅ |
| `getAll()` (lowercase) | Same package only | External call ❌ |

**Rule of thumb:** If another package needs it → **capitalize it**.

### 4. Importing Your Own Packages

```go
package main

import (
    // Standard library
    "fmt"
    "log"

    // Third-party (from go.mod)
    _ "github.com/mattn/go-sqlite3"

    // YOUR OWN PACKAGES — module path + directory path
    "github.com/YOURNAME/go-sql-learning/model"
    "github.com/YOURNAME/go-sql-learning/repository"
    "github.com/YOURNAME/go-sql-learning/handler"
)
```

The import path is always: **`<module-path-from-go-mod>/<directory-name>`**

---

## Phase A: Multi-File, Same Package (Warm-Up)

**Goal:** Split `main.go` into multiple files without changing any behavior.
**Difficulty:** Easiest — same `package main`, just different files.

### Target Structure (Phase A only)

```
project/
├── go.mod
├── main.go          ← InitDB, CloseDB, func main()
├── model.go         ← Todo struct, LIMIT constant
├── repository.go    ← TodoRepo, NewTodoRepo, all CRUD methods
└── handler.go       ← PrintHelp, all handle* functions
```

### Step A1: Create `model.go`

```go
package main

import "time"

type Todo struct {
	ID        int
	Task      string
	Done      bool
	CreatedAt time.Time
}

const LIMIT = 10
```

**What changed from `main.go`:**
- Removed from `main.go`
- Same `package main` → everything still works together
- `Todo` and `LIMIT` are accessible from all other files in `package main`

### Step A2: Create `repository.go`

Take ALL these functions from `main.go` and put them here (still `package main`):

```go
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
	err := r.db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&total)
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
```

### Step A3: Create `handler.go`

```go
package main

import (
	"fmt"
	"strconv"
	"strings"
)

func PrintHelp() {
	fmt.Println("\n Available Commands:")
	fmt.Println("  help | h              Show help")
	fmt.Println("  list <page>           List task (default page = 1)")
	fmt.Println("  create <task>         Create task")
	fmt.Println("  delete <id>           Delete task by ID")
	fmt.Println("  update <id> <task>    Update task")
	fmt.Println("  toggle  <id>          Toggle task")
	fmt.Println("  version | v           Show version")
}

func handleList(repo *TodoRepo, args []string) {
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
	todos, total, err := repo.GetTodos(page)
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

func handleCreate(repo *TodoRepo, args []string) {
	if len(args) < 2 {
		fmt.Println("task text is required")
		fmt.Println("Usage: create <task>")
		return
	}
	task := strings.Join(args[1:], " ")
	todo, err := repo.Create(task)
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

func handleDelete(repo *TodoRepo, args []string) {
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
	rowsAffected, err := repo.Delete(id)
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

func handleUpdate(repo *TodoRepo, args []string) {
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
	todo, err := repo.Update(id, task)
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

func handleToggle(repo *TodoRepo, args []string) {
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
	todo, err := repo.Toggle(id)
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
```

### Step A4: Slim Down `main.go`

After moving code out, `main.go` becomes tiny:

```go
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
```

### Test Phase A

```bash
go run .                    # compiles all files in package main
go run . list               # test listing
go run . create "buy milk"  # test creating
go run . list               # verify it persisted
```

**If it works, you've successfully split one file into 4 files. Zero behavior change.**

---

## Phase B: Extract Model Package

**Goal:** Move `Todo` struct into its own package so other packages can share it.
**Difficulty:** Medium — now you're dealing with cross-package visibility.

### What Changes

| Before (Phase A) | After (Phase B) |
|------------------|-----------------|
| `type Todo struct {...}` in `model.go` (`package main`) | `type Todo struct {...}` in `model/todo.go` (`package model`) |
| Used as `Todo` everywhere | Used as `model.Todo` in other packages |

### Step B1: Create `model/todo.go`

```bash
mkdir -p model
```

```go
package model

import "time"

// Todo represents a single todo item.
// Exported (uppercase T) because other packages need it.
type Todo struct {
	ID        int
	Task      string
	Done      bool
	CreatedAt time.Time
}

// LIMIT is the default pagination size.
// Exported because handlers in other packages need it.
const LIMIT = 10
```

### Step B2: Update ALL Other Files

Every file that used `Todo` or `LIMIT` now needs:
1. Add `"github.com/YOURNAME/go-sql-learning/model"` to imports
2. Change `Todo` → `model.Todo`
3. Change `LIMIT` → `model.LIMIT` (or keep using `LIMIT` if you import with `.` — but don't do that yet)

**In `repository.go`, change:**
```go
// BEFORE
func (r *TodoRepo) Create(task string) (Todo, error) {
    return Todo{}, err
}

// AFTER
import "github.com/YOURNAME/go-sql-learning/model"

func (r *TodoRepo) Create(task string) (model.Todo, error) {
    return model.Todo{}, err
}
```

**In `handler.go`, change:**
```go
import "github.com/YOURNAME/go-sql-learning/model"

// All Todo references become model.Todo
// All LIMIT references become model.LIMIT
```

**In `main.go`:**
- `main.go` might not directly use `Todo` anymore (handlers/repo handle that), but if it does → `model.Too`

### Step B3: Delete `model.go`

Remove the old `model.go` from the root — its contents now live in `model/todo.go`.

---

## Phase C: Extract Repository Package

**Goal:** Move `TodoRepo`, interface, and all CRUD methods into their own package.
**Difficulty:** Medium-High — handlers will now depend on an imported package.

### Step C1: Create Directory and Files

```bash
mkdir -p repository
```

### Step C2: Create `repository/repo.go`

```go
package repository

import (
	"database/sql"
	"github.com/YOURNAME/go-sql-learning/model"
)

// TodoRepository defines the contract for todo data access.
// Any struct with these methods satisfies this interface automatically.
type TodoRepository interface {
	Create(task string) (model.Todo, error)
	GetTodos(page int) ([]model.Todo, int, error)
	Update(id int, task string) (model.Todo, error)
	Toggle(id int) (model.Todo, error)
	Delete(id int) (int64, error)
}

// TodoRepo is the SQLite implementation of TodoRepository.
type TodoRepo struct {
	db *sql.DB
}

// NewTodoRepo creates a new TodoRepo wrapping the given DB connection.
func NewTodoRepo(db *sql.DB) *TodoRepo {
	return &TodoRepo{db: db}
}
```

### Step C3: Create `repository/crud.go`

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/YOURNAME/go-sql-learning/model"
)

func (r *TodoRepo) Create(task string) (model.Todo, error) {
	insertSQL := `INSERT INTO todos (task) VALUES (?)`
	result, err := r.db.Exec(insertSQL, task)
	if err != nil {
		return model.Todo{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return model.Todo{}, err
	}
	var todo model.Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return model.Todo{}, err
	}
	return todo, nil
}

func (r *TodoRepo) GetTodos(page int) ([]model.Todo, int, error) {
	var todos []model.Todo
	offset := (page - 1) * model.LIMIT
	queryAllSQL := `SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`
	rows, err := r.db.Query(queryAllSQL, model.LIMIT, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var todo model.Todo
		if err := rows.Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt); err != nil {
			return nil, 0, err
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&total)
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

func (r *TodoRepo) Update(id int, task string) (model.Todo, error) {
	updateSQL := `UPDATE todos SET task = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.Exec(updateSQL, task, id)
	if err != nil {
		return model.Todo{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.Todo{}, err
	}
	if rowsAffected == 0 {
		return model.Todo{}, fmt.Errorf("todo with ID %d not found", id)
	}
	var todo model.Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return model.Todo{}, err
	}
	return todo, nil
}

func (r *TodoRepo) Toggle(id int) (model.Todo, error) {
	updateSQL := `UPDATE todos SET done = NOT done, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.Exec(updateSQL, id)
	if err != nil {
		return model.Todo{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.Todo{}, err
	}
	if rowsAffected == 0 {
		return model.Todo{}, fmt.Errorf("todo with ID %d not found", id)
	}
	var todo model.Todo
	querySQL := `SELECT id, task, done, created_at FROM todos WHERE id = ?`
	err = r.db.QueryRow(querySQL, id).Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
	if err != nil {
		return model.Todo{}, err
	}
	return todo, nil
}
```

### Step C4: Update `handler.go`

Handlers now accept `repository.TodoRepository` instead of `*TodoRepo`:

```go
package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/YOURNAME/go-sql-learning/model"
	"github.com/YOURNAME/go-sql-learning/repository"
)

func PrintHelp() {
	fmt.Println("\n Available Commands:")
	fmt.Println("  help | h              Show help")
	fmt.Println("  list <page>           List task (default page = 1)")
	fmt.Println("  create <task>         Create task")
	fmt.Println("  delete <id>           Delete task by ID")
	fmt.Println("  update <id> <task>    Update task")
	fmt.Println("  toggle  <id>          Toggle task")
	fmt.Println("  version | v           Show version")
}

func handleList(svc repository.TodoRepository, args []string) {
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
	todos, total, err := svc.GetTodos(page)
	if err != nil {
		fmt.Println("Error getting the todos", err)
		return
	}
	totalPages := (total + model.LIMIT - 1) / model.LIMIT
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

func handleCreate(svc repository.TodoRepository, args []string) {
	if len(args) < 2 {
		fmt.Println("task text is required")
		fmt.Println("Usage: create <task>")
		return
	}
	task := strings.Join(args[1:], " ")
	todo, err := svc.Create(task)
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

func handleDelete(svc repository.TodoRepository, args []string) {
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
	rowsAffected, err := svc.Delete(id)
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

func handleUpdate(svc repository.TodoRepository, args []string) {
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
	todo, err := svc.Update(id, task)
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

func handleToggle(svc repository.TodoRepository, args []string) {
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
	todo, err := svc.Toggle(id)
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
```

**Notice:** `handle*` functions now take `repository.TodoRepository` (the **interface**) — this is Approach 3 applied across packages!

### Step C5: Final `main.go`

```go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/YOURNAME/go-sql-learning/handler"
	"github.com/YOURNAME/go-sql-learning/repository"

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
		handler.PrintHelp()
		return
	}

	repo := repository.NewTodoRepo(db)

	switch args[0] {
	case "help", "h":
		handler.PrintHelp()
	case "list":
		handler.HandleList(repo, args)
	case "create":
		handler.HandleCreate(repo, args)
	case "delete":
		handler.HandleDelete(repo, args)
	case "update":
		handler.HandleUpdate(repo, args)
	case "toggle":
		handler.HandleToggle(repo, args)
	case "version", "v":
		fmt.Println("App Version: 1.0.0")
	default:
		fmt.Println("Unknown command")
		handler.PrintHelp()
	}
}
```

**Notice the changes:**
- Imports `handler` and `repository` packages
- Calls `handler.HandleList()` (capital H = exported!)
- Calls `handler.PrintHelp()` (moved out of main)
- `main()` is ~50 lines — just wiring things together

---

## Final Project Structure (Phase C Complete)

```
go-sql-learning/
├── go.mod                              ← module definition
├── main.go                             ← entry point (~60 lines)
├── model/
│   └── todo.go                         ← Todo struct + LIMIT (~12 lines)
├── repository/
│   ├── repo.go                         ← Interface + struct + constructor (~20 lines)
│   └── crud.go                         ← All CRUD methods (~150 lines)
└── handler/
    └── handler.go                      ← PrintHelp + all handle* functions (~170 lines)
```

**Dependency flow (no cycles!):**
```
main ──→ handler ──→ repository ──→ model
  │                   ↑                ↑
  └──→ repository ───┘                │
  └──→ model ─────────────────────────┘
```

**Rule:** Dependencies point **downward**. `model` knows about nothing. `repository` knows about `model`. `handler` knows about `repository` + `model`. `main` knows about everyone but nobody imports `main`.

---

## Essential Go Commands Reference

| Command | What It Does |
|---------|-------------|
| `go mod init <path>` | Create `go.mod` (do once) |
| `go build ./...` | Build all packages (check for compile errors) |
| `go run .` | Compile + run `main` package (all files) |
| `go vet ./...` | Catch suspicious code patterns |
| `go fmt ./...` | Format all code per Go standards |
| `go list -m all` | List all dependencies |
| `go mod tidy` | Clean up unused imports in go.mod |
| `go test ./...` | Run all tests in all packages |

---

## Common Pitfalls (And How to Fix Them)

### 1. "Cannot refer to unexported name"

```go
// In package model:
type todo struct { ... }  // lowercase = private!

// In package main:
var t model.todo  // ❌ ERROR: cannot refer to unexported name
```

**Fix:** Capitalize → `type Todo struct { ... }`

### 2. Import Cycle Not Allowed

```
main imports repository
repository imports main    ← CYCLE! Go forbids this
```

**Fix:** Never import `main` from any other package. If `main` needs something from another package, that's fine — but never the reverse.

### 3. Wrong Import Path

```
cannot find package: "model/todo"
```

**Fix:** Use the full module path: `"github.com/YOURNAME/go-sql-learning/model"`

### 4. Forgot `package` Declaration

```
expected 'package', found 'type'
```

**Fix:** Every `.go` file MUST start with `package <name>`.

### 5. Duplicate Type Declarations

```
Todo redeclared in this block
```

**Fix:** After extracting `Todo` to `model/todo.go`, remove the old declaration from other files.

### 6. Using Unexported Field Across Packages

```go
// model/todo.go
type Todo struct {
    id int       // lowercase = unexported!
}

// repository/crud.go
todo.ID  // ❌ ERROR: ID is unexported
```

**Fix:** Struct fields that need to be accessed from other packages must also be capitalized.

---

## Package Naming Conventions

| Convention | Example | When to Use |
|-----------|---------|-------------|
| Lowercase, single word | `model`, `handler`, `repository` | Most packages |
| Descriptive noun | `storage`, `auth`, `middleware` | Functionality packages |
| Never: `models`, `helpers`, `utils` | — | Avoid plural names and generic "util" dumps |
| `package main` | entry point | Only for executables |

---

## Practice Checklist

Complete each phase and test before moving on:

- [ ] **Phase A**: Split into 4 files (all `package main`). Run `go run .` + test all commands.
- [ ] **Phase B**: Extract `model/` package. Run `go build ./...`. Fix all `Todo` → `model.Todo`.
- [ ] **Phase C**: Extract `repository/` package. Run `go build ./...`. Fix all imports.
- [ ] **Phase D**: Extract `handler/` package. Run `go build ./...`. Fix handler calls in `main`.
- [ ] **Final test**: Run `go run .` with all commands (create, list, update, toggle, delete).
- [ ] **Formatting**: Run `go fmt ./...` — see Go reformat your code.
- [ ] **Vetting**: Run `go vet ./...` — catch any issues.
