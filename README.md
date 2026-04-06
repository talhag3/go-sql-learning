# go-sqlite-learning

A simple CLI Todo application built with Go and SQLite to practice database abstraction and CRUD operations.

## Features

- Create, Read, Update, Delete (CRUD) todos
- Toggle todo completion status
- Paginated list view
- Persistent storage with SQLite
- Clean database abstraction layer

## Requirements

- Go 1.21+
- SQLite3

## Installation

```bash
git clone <repository-url>
cd go-sqlite-learning
go mod download
```

## Usage

```bash
go run main.go <command> [arguments]
```

### Available Commands

| Command | Arguments | Description |
|---------|-----------|-------------|
| `help`, `h` | - | Show help menu |
| `list` | `[page]` | List todos (default: page 1, 10 per page) |
| `create` | `<task>` | Create a new todo |
| `update` | `<id> <task>` | Update a todo's task text |
| `toggle` | `<id>` | Toggle todo completion status |
| `delete` | `<id>` | Delete a todo by ID |
| `version`, `v` | - | Show application version |

### Examples

```bash
# Create a new todo
go run main.go create Buy groceries

# Create todo with multiple words
go run main.go create "Learn Go database/sql package"

# List todos (first page)
go run main.go list

# List todos (page 2)
go run main.go list 2

# Update a todo
go run main.go update 1 "Buy groceries and cook dinner"

# Toggle completion status
go run main.go toggle 1

# Delete a todo
go run main.go delete 1
```

## Project Structure

```
.
├── main.go      # Application code (single file)
├── todos.db     # SQLite database (auto-created)
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- `database/sql` - Go standard library for database operations
- `github.com/mattn/go-sqlite3` - SQLite3 driver

## Database Schema

```sql
CREATE TABLE todos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task TEXT NOT NULL,
    done BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Learning Objectives

This project demonstrates:

- Database connection management
- SQL query execution with parameterized statements
- CRUD operations abstraction
- CLI argument parsing
- Error handling patterns
- Resource cleanup with `defer`

## Future Improvements

- [ ] Split code into multiple files (db.go, handlers.go, models.go)
- [ ] Add configuration file support