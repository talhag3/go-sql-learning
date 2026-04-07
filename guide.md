# Guide: Refactoring DB Access — From Beginner to Idiomatic Go

## Table of Contents

1. [The Problem You Identified](#1-the-problem-you-identified)
2. [Go Prerequisites You Need First](#2-go-prerequisites-you-need-first)
3. [Approach 1 — Package-Level Variable](#3-approach-1--package-level-variable)
4. [Approach 2 — Struct with Methods (Recommended)](#4-approach-2--struct-with-methods-recommended)
5. [Approach 3 — Interface + Dependency Injection (Advanced)](#5-approach-3--interface--dependency-injection-advanced)
6. [Approach 4 — Multi-File & Multi-Package Architecture](#6-approach-4--multi-file--multi-package-architecture)
7. [Comparison Summary](#7-comparison-summary)
8. [Your Action Plan](#8-your-action-plan)

---

## 1. The Problem You Identified

Right now your code looks like this flow:

```
main()
  └─→ handleCreate(db, args)
        └─→ createTask(db, task)    ← db passed again
```

Every handler receives `db`, then passes it again to the data function. This is called **"threading a dependency"** — you're manually passing the same value through multiple layers.

**Why it feels wrong:**
- Every new function you add needs `db` as its first parameter
- If you ever need a second dependency (e.g., a logger), every function signature changes
- The caller has to know about `db` even when they only care about "get me todos"
- It mixes **"how"** (database connection) with **"what"** (business logic)

**What we want:**
```
repo.GetTodos(page)     ← caller doesn't know or care about db
repo.Create("buy milk") ← clean, self-documenting
```

---

## 2. Go Prerequisites You Need First

Before refactoring, make sure you understand these concepts. If any are unclear, research them first.

### 2.1 Structs vs Classes (Mental Model Shift)

If you come from Python, Java, or JavaScript — **Go does not have classes**. It has **structs** (data) and **methods** (functions attached to structs).

```go
// This is JUST data — no behavior attached yet
type User struct {
    Name string
    Age  int
}

// This is a METHOD — a function bound to User
// The (u *User) part is called the "receiver"
func (u *User) Greet() string {
    return "Hello, my name is " + u.Name
}

// Usage
u := &User{Name: "Ali", Age: 25}
fmt.Println(u.Greet())  // Hello, my name is Ali
```

**Key terms:**
| Term | Meaning |
|------|---------|
| `struct` | A collection of fields (data only) |
| `method` | A function with a receiver `(x *Struct)` |
| `receiver` | The `(r *Repo)` part — like `self` in Python or `this` in Java |
| `*` before type | Pointer receiver — lets the method modify the struct's fields |

### 2.2 Value Receiver vs Pointer Receiver

```go
type Counter struct {
    count int
}

// Value receiver — works on a COPY of the struct
func (c Counter) IncrementValue() {
    c.count++  // modifies the copy, original unchanged!
}

// Pointer receiver — works on the ORIGINAL struct
func (c *Counter) IncrementPointer() {
    c.count++  // modifies the original
}
```

**Rule of thumb:** Use **pointer receiver** (`*Type`) when:
- The method needs to modify the struct's fields
- The struct is large (avoid copying)
- You want consistency — if one method uses pointer, all should

For our `TodoRepo`, we'll use **pointer receiver** because we need to access `r.db`.

### 2.3 Constructor Pattern in Go

Go has no built-in constructor keyword. Instead, the convention is a `NewXxx()` function:

```go
type Database struct {
    connection string
}

// Constructor — returns a pointer to the initialized struct
func NewDatabase(conn string) *Database {
    return &Database{connection: conn}
}

// Usage
db := NewDatabase("localhost:5432")
```

This is just a regular function — the `New` prefix is a **community convention**, not a language rule.

### 2.4 Package-Level Variables

In Go, variables declared outside of functions (at package level) are accessible from any function in that package:

```go
package main

var globalCount int = 0  // package-level variable

func increment() {
    globalCount++  // can access directly
}

func printCount() {
    fmt.Println(globalCount)  // can access directly
}
```

Lowercase (`globalCount`) = **unexported** (only visible within the same package).  
Uppercase (`GlobalCount`) = **exported** (visible from other packages).

---

## 3. Approach 1 — Package-Level Variable

### The Idea

Store `db` as a package-level variable. All functions access it directly — no need to pass it as a parameter.

### Concept Diagram

```
BEFORE (your current code):
  main → handleCreate(db) → createTask(db, task)

AFTER (package-level var):
  main → handleCreate(args) → createTask(task)
                                    ↑
                              uses package-level db directly
```

### Code Structure

```go
package main

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

// PACKAGE-LEVEL VARIABLE — visible to all functions in this package
var db *sql.DB

func InitDB() {
    var err error
    db, err = sql.Open("sqlite3", "todos.db")
    if err != nil {
        log.Fatal(err)
    }
    // ... rest of init logic
}

// No more db parameter! Uses the package-level variable directly.
func getTodos(page int) ([]Todo, int, error) {
    query := `SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`
    rows, err := db.Query(query, LIMIT, offset)  // ← uses package-level db
    // ...
}

func createTask(task string) (Todo, error) {
    result, err := db.Exec(`INSERT INTO todos (task) VALUES (?)`, task)  // ← same
    // ...
}
```

### Why People Use This
- Simplest change — just remove parameters
- Works fine for small scripts and CLI tools
- Low mental overhead

### Why It's Not Ideal for Growth
| Problem | Explanation |
|---------|-------------|
| **Hidden dependency** | A function like `getTodos(1)` looks like it needs nothing, but secretly depends on `db` being initialized. A reader can't tell from the signature. |
| **Hard to test** | You can't easily pass a test database or mock. Tests would need to set the global `db` before each test, which is fragile. |
| **Shared mutable state** | Any function in the package can modify `db`. As the program grows, this causes bugs that are hard to trace. |
| **Only one DB ever** | What if you later want two database connections? You can't — there's only one global slot. |
| **Initialization order risk** | If something calls `getTodos` before `InitDB()` runs, you get a nil pointer panic. No compiler protection. |

### When Is This OK?
- Tiny single-file scripts where you'll never add tests
- Quick prototypes you'll throw away
- Learning the absolute basics (you're past this stage now!)

---

## 4. Approach 2 — Struct with Methods (Recommended)

### The Idea

Wrap `db` inside a **struct**. Make all data functions **methods** on that struct. Create the struct once in `main()`, pass the **struct** (not raw `db`) to handlers.

### Concept Diagram

```
BEFORE:
  handleCreate(db, args) → createTask(db, task)

AFTER:
  repo := NewTodoRepo(db)       ← wrap db in a struct ONCE
  handleCreate(repo, args)      ← pass the repo (which holds db internally)
      └→ repo.Create(task)      ← method call, db is inside repo
```

### Step-by-Step Implementation Guide

#### Step 1: Define the Repository Struct

```go
// TodoRepo holds the database connection and provides data access methods.
// Think of it as "an object that knows how to talk to the todos table".
type TodoRepo struct {
    db *sql.DB
}
```

**Why call it "Repo"?**
- Short for **Repository** — a common pattern meaning "an object that handles all DB operations for one entity"
- Other common names: `Service`, `Store`, `DAO` (Data Access Object)
- The name signals intent: this struct's job is **data access**, not business logic

#### Step 2: Write the Constructor

```go
// NewTodoRepo creates a new TodoRepo wrapping the given database connection.
// This is the ONLY place where db is assigned to the struct.
func NewTodoRepo(db *sql.DB) *TodoRepo {
    return &TodoRepo{db: db}
}
```

**Why a constructor instead of direct initialization?**
- Validation could be added here later (e.g., check db isn't nil)
- It's the standard Go pattern — developers expect `NewXxx()` functions
- If you later add fields (like a logger), you initialize them all in one place

#### Step 3: Convert Functions to Methods (Example: GetTodos)

**Before (current function):**

```go
func getTodos(db *sql.DB, page int) ([]Todo, int, error) {
    offset := (page - 1) * LIMIT
    rows, err := db.Query(queryAllSQL, LIMIT, offset)
    // ...
}
```

**After (method on TodoRepo):**

```go
// GetTodos returns a paginated list of todos and the total count.
// Notice: no db parameter! It uses r.db (the struct's field).
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error) {
    var todos []Todo
    offset := (page - 1) * LIMIT

    queryAllSQL := `SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`
    rows, err := r.db.Query(queryAllSQL, LIMIT, offset)   // ← r.db, not db
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
```

**What changed:**
- `func getTodos(db *sql.DB, page int)` → `func (r *TodoRepo) GetTodos(page int)`
- Every `db.Query(...)` → `r.db.Query(...)`
- That's it. Same logic, cleaner signature.

#### Step 4: Do the Same for All Other Functions

Here's a cheat sheet showing the transformation for each function:

| Current Function Signature | New Method Signature |
|---|---|
| `func createTask(db *sql.DB, task string)` | `func (r *TodoRepo) Create(task string)` |
| `func getTodos(db *sql.DB, page int)` | `func (r *TodoRepo) GetTodos(page int)` |
| `func deleteTask(db *sql.DB, id int)` | `func (r *TodoRepo) Delete(id int)` |
| `func updateTask(db *sql.DB, id int, task string)` | `func (r *TodoRepo) Update(id int, task string)` |
| `func toggleTask(db *sql.DB, id int)` | `func (r *TodoRepo) Toggle(id int)` |

**Naming convention change notice:** In Go, **exported names** (uppercase first letter) are public. Since these methods might be used from other packages someday, we capitalize them: `Create` not `create`, `GetTodos` not `getTodos`. This is idiomatic Go.

#### Step 5: Update Handlers to Use Repo

**Before:**

```go
func handleCreate(db *sql.DB, args []string) {
    task := strings.Join(args[1:], " ")
    todo, err := createTask(db, task)   // ← passing db
    // ...
}
```

**After:**

```go
func handleCreate(repo *TodoRepo, args []string) {
    task := strings.Join(args[1:], " ")
    todo, err := repo.Create(task)   // ← no db, method call on repo
    // ...
}
```

**Same pattern for all handlers:**
- Replace `db *sql.DB` parameter with `repo *TodoRepo`
- Change `functionName(db, ...)` to `repo.MethodName(...)`

#### Step 6: Update main()

**Before:**

```go
func main() {
    db := InitDB()
    defer CloseDB(db)
    // ...
    case "create":
        handleCreate(db, args)
}
```

**After:**

```go
func main() {
    db := InitDB()
    defer CloseDB(db)

    repo := NewTodoRepo(db)   // ← create repo once

    switch args[0] {
    case "create":
        handleCreate(repo, args)   // ← pass repo, not db
    case "list":
        handleList(repo, args)     // ← same
    // ... etc
    }
}
```

### Why This Is Better

| Benefit | Explanation |
|---------|-------------|
| **Explicit dependency** | `handleCreate(repo, args)` tells you: this function needs a TodoRepo. No hidden globals. |
| **Self-contained** | `TodoRepo` owns its `db`. All data logic lives in one place. |
| **Easy to test** | You can create `NewTodoRepo(testDB)` with a test database. No global state to reset. |
| **Scalable** | Add a logger field to `TodoRepo` later without changing method signatures. |
| **Idiomatic Go** | This is how real Go projects are structured. You're learning real patterns. |
| **Encapsulation** | Callers use `repo.Create(...)` — they don't need to know about SQL or `*sql.DB`. |

### Full Skeleton (What You're Building Toward)

```go
package main

// --- MODEL ---
type Todo struct { /* ... */ }

// --- REPOSITORY ---
type TodoRepo struct { db *sql.DB }

func NewTodoRepo(db *sql.DB) *TodoRepo { /* ... */ }

func (r *TodoRepo) Create(task string) (Todo, error)         { /* uses r.db */ }
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error)   { /* uses r.db */ }
func (r *TodoRepo) Update(id int, task string) (Todo, error)  { /* uses r.db */ }
func (r *TodoRepo) Toggle(id int) (Todo, error)              { /* uses r.db */ }
func (r *TodoRepo) Delete(id int) (int64, error)              { /* uses r.db */ }

// --- HANDLERS ---
func handleList(repo *TodoRepo, args []string)     { /* uses repo.GetTodos() */ }
func handleCreate(repo *TodoRepo, args []string)   { /* uses repo.Create() */ }
func handleUpdate(repo *TodoRepo, args []string)   { /* uses repo.Update() */ }
func handleToggle(repo *TodoRepo, args []string)   { /* uses repo.Toggle() */ }
func handleDelete(repo *TodoRepo, args []string)   { /* uses repo.Delete() */ }

// --- MAIN ---
func main() {
    db := InitDB()
    defer CloseDB(db)
    repo := NewTodoRepo(db)
    // route commands to handlers, passing repo
}
```

---

## 5. Approach 3 — Interface + Dependency Injection (Advanced)

### The Idea

Define an **interface** that lists what operations are needed. Your handlers depend on the **interface**, not the concrete struct. This means you can swap in a fake implementation for testing.

### Concept Diagram

```
                    ┌─────────────────────┐
                    │   TodoRepository    │  ← interface (contract)
                    │  - Create()         │
                    │  - GetTodos()       │
                    │  - Update()         │
                    │  - Toggle()         │
                    │  - Delete()         │
                    └──────────┬──────────┘
                               │ implements
              ┌────────────────┼────────────────┐
              ▼                                 ▼
   ┌──────────────────┐               ┌──────────────────┐
   │   TodoRepo       │               │   MockTodoRepo   │
   │  (real SQLite)   │               │  (fake for tests)│
   └──────────────────┘               └──────────────────┘
```

### Code Structure

```go
// STEP 1: Define the interface
type TodoRepository interface {
    Create(task string) (Todo, error)
    GetTodos(page int) ([]Todo, int, error)
    Update(id int, task string) (Todo, error)
    Toggle(id int) (Todo, error)
    Delete(id int) (int64, error)
}

// STEP 2: Concrete implementation (same as Approach 2)
type TodoRepo struct { db *sql.DB }
func NewTodoRepo(db *sql.DB) *TodoRepo { return &TodoRepo{db: db} }
// ... methods same as before ...

// STEP 3: Handlers accept the INTERFACE, not the struct
func handleCreate(svc TodoRepository, args []string) {
    todo, err := svc.Create(task)  // works with ANY implementation
}

// STEP 4: In main(), pass the real implementation
func main() {
    repo := NewTodoRepo(db)
    handleCreate(repo, args)  // TodoRepo satisfies TodoRepository automatically
}
```

**Go interfaces are implicit** — you don't write `implements` like in Java. If a struct has all the methods an interface defines, Go automatically considers it an implementer. This is called **"structural typing"**.

### When to Use This
- Writing unit tests with mocks/fakes
- You have multiple implementations (SQLite repo, PostgreSQL repo, in-memory repo)
- Building a library where users should supply their own implementation

### When NOT to Use This (Yet)
- You don't have tests yet
- You only have one database implementation
- It adds abstraction you don't currently need

**Principle:** "Prefer concrete types until the abstraction proves necessary." Don't over-engineer early.

---

## 6. Approach 4 — Multi-File & Multi-Package Architecture

### The Idea

Your code lives in a **single 400-line `main.go`** file. Real Go projects split code across **multiple files and packages**, each with a clear responsibility. This is where you learn Go's module system, package visibility rules, and project organization.

### What You'll Learn

| Concept | Why It Matters |
|---------|---------------|
| **Go Modules** (`go.mod`) | How Go manages dependencies and your project identity |
| **Packages** | How to group related code together |
| **Multi-file same package** | Split one big file into logical files without changing anything |
| **Multi-package** | Separate concerns (models, database, handlers) into their own packages |
| **Exported vs Unexported** | Uppercase = public (cross-package), lowercase = private (same package only) |
| **Importing local packages** | How to use your own packages with `import` |
| **Project layout conventions** | How real Go projects are structured |

### Concept Diagram

```
CURRENT (single file):
┌─────────────────────────────────────┐
│           main.go (402 lines)       │
│  ┌──────┬──────┬──────┬──────────┐  │
│  │Model │ DB   │Repo  │Handlers  │  │
│  │Todo  │Init  │CRUD  │handle*   │  │
│  │struct│Close │funcs │main()    │  │
│  └──────┴──────┴──────┴──────────┘  │
└─────────────────────────────────────┘

AFTER (multi-package):
go-sql-learning/
├── go.mod
├── main.go                    ← entry point only (wire things together)
├── model/
│   └── todo.go                ← Todo struct
├── repository/
│   ├── repo.go                ← TodoRepo struct + interface + methods
│   └── sqlite.go              ← SQLite-specific implementation
└── handler/
    ├── handler.go             ← all handle* functions
    └── help.go                ← PrintHelp, version info
```

### Prerequisites: Key Concepts

#### What is a Go Module?

A **module** is a collection of Go packages that are released together. It's identified by a **module path** (e.g., `github.com/yourname/go-sql-learning`). The `go.mod` file at your project root defines it:

```bash
$ cat go.mod
module github.com/yourname/go-sql-learning

go 1.21
```

If you don't have a `go.mod` yet, create one:
```bash
go mod init github.com/yourname/go-sql-learning
```

#### What is a Package?

A **package** is a directory containing `.go` files that all declare `package <name>`. Packages are the unit of **reuse** and **visibility** in Go:

```go
// model/todo.go
package model     // ← directory name = package name (convention)

type Todo struct { ... }   // ← unexported: only visible inside "model" package
```

#### Visibility Rules (Critical!)

This is the #1 thing that trips up beginners moving from single-file to multi-package:

| Name | Visible From | Example |
|------|-------------|---------|
| `Todo` (uppercase) | **Any package** that imports `model` | `model.Todo{}` ✅ |
| `todo` (lowercase) | **Only** the same package (`model`) only | `model.todo{}` ❌ compile error |
| `GetTodos` (uppercase) | Any importing package | `repo.GetTodos(1)` ✅ |
| `getTodos` (lowercase) | Same package only | External access ❌ |

**Rule:** Uppercase first letter = **exported** (public). Lowercase = **unexported** (private).

#### Importing Your Own Packages

When you split code into packages, you import them using the **module path + package path**:

```go
// main.go
package main

import (
    "github.com/yourname/go-sql-learning/model"      // your own package!
    "github.com/yourname/go-sql-learning/repository" // your own package!
)

func main() {
    // Use exported names only
    var t model.Todo          // ✅ Todo is exported
    repo := repository.NewTodoRepo(db)  // ✅ NewTodoRepo is exported
}
```

---

### Step-by-Step Implementation

#### Phase A: Multi-File, Same Package (Easiest First)

Before creating new packages, just split `main.go` into multiple files — all still in `package main`. **Zero behavior change**, just organization.

**Target structure:**
```
project/
├── main.go          ← main() + InitDB/CloseDB + imports
├── model.go         ← Todo struct + LIMIT constant
├── repository.go    ← TodoRepo, NewTodoRepo, all CRUD methods
└── handler.go       ← PrintHelp + all handle* functions
```

**How:** Create each file with `package main` at the top. Move the relevant code. That's it. Go automatically compiles all `.go` files in the same package together.

**Rules for same-package multi-file:**
- All files: `package main`
- No duplicate declarations (e.g., don't define `Todo` in two files)
- One file has `func main()`, others don't
- Run with: `go run .` (compiles all files)

#### Phase B: Extract Model Package

Move the `Todo` struct into its own package so other packages can import it without depending on `main`.

**Create `model/todo.go`:**
```go
package model

import "time"

type Todo struct {
    ID        int
    Task      string
    Done      bool
    CreatedAt time.Time
}
```

**Update `main.go` (and any other file that uses `Todo`):**
```go
package main

import "github.com/yourname/go-sql-learning/model"

// Now use model.Todo instead of Todo
func createTask(db *sql.DB, task string) (model.Todo, error) {
    // ...
    return model.Todo{}, err
}
```

**Key changes:**
- Every `Todo` → `model.Todo`
- The `LIMIT` constant can move here too if other packages need it

#### Phase C: Extract Repository Package

Move `TodoRepo`, the interface, and all CRUD methods into their own package.

**Create `repository/repo.go`:**
```go
package repository

import (
    "database/sql"
    "github.com/yourname/go-sql-learning/model"
)

// INTERFACE — defines the contract
type TodoRepository interface {
    Create(task string) (model.Todo, error)
    GetTodos(page int) ([]model.Todo, int, error)
    Update(id int, task string) (model.Todo, error)
    Toggle(id int) (model.Todo, error)
    Delete(id int) (int64, error)
}

// CONCRETE IMPLEMENTATION
type TodoRepo struct {
    db *sql.DB
}

func NewTodoRepo(db *sql.DB) *TodoRepo {
    return &TodoRepo{db: db}
}

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
    err = r.db.QueryRow(`SELECT id, task, done, created_at FROM todos WHERE id = ?`, id).
        Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt)
    return todo, err
}

func (r *TodoRepo) GetTodos(page int) ([]model.Todo, int, error) {
    const LIMIT = 10
    offset := (page - 1) * LIMIT
    rows, err := r.db.Query(`SELECT id, task, done, created_at FROM todos LIMIT ? OFFSET ?`, LIMIT, offset)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()
    var todos []model.Todo
    for rows.Next() {
        var todo model.Todo
        if err := rows.Scan(&todo.ID, &todo.Task, &todo.Done, &todo.CreatedAt); err != nil {
            return nil, 0, err
        }
        todos = append(todos, todo)
    }
    var total int
    err := r.db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&total)
    return todos, total, err
}

// ... Update, Toggle, Delete follow the same pattern
```

**Update `main.go`:**
```go
package main

import (
    "github.com/yourname/go-sql-learning/model"
    "github.com/yourname/go-sql-learning/repository"
)

func main() {
    db := InitDB()
    defer CloseDB(db)

    repo := repository.NewTodoRepo(db)  // ← imported package
    // ...
}
```

#### Phase D: Slim Down main.go

After extraction, `main.go` should be **only** wiring things together — no business logic:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    "github.com/yourname/go-sql-learning/model"
    "github.com/yourname/go-sql-learning/repository"
)

func main() {
    args := os.Args[1:]
    db := InitDB()
    defer CloseDB(db)

    if len(args) == 0 {
        fmt.Println("No command provided")
        PrintHelp()
        return
    }

    repo := repository.NewTodoRepo(db)

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

---

### Final Project Structure

```
go-sql-learning/
├── go.mod                              ← module definition
├── main.go                             ← entry point (~50 lines)
├── model/
│   └── todo.go                         ← Todo struct (~10 lines)
├── repository/
│   ├── repo.go                         ← TodoRepo, interface, constructor (~20 lines)
│   └── crud.go                         ← Create, GetTodos, Update, Toggle, Delete (~150 lines)
└── handler/
    ├── handler.go                      ← handleList, handleCreate, handleDelete (~100 lines)
    ├── handler_update_toggle.go        ← handleUpdate, handleToggle (~80 lines)
    └── help.go                         ← PrintHelp, version (~15 lines)
```

### Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| Lowercase name used from another package | `cannot refer to unexported name` | Capitalize the name (`todo` → `Todo`) |
| Circular imports | `import cycle not allowed` | Re-think which package depends on which (e.g., both `model` and `repository` should never import `main`) |
| Forgot `package` declaration | `expected 'package'` | Every `.go` file must start with `package <name>` |
| Wrong import path | `cannot find package` | Use module path from `go.mod` + relative path to directory |
| Duplicate type name across packages | Confusion about which `Todo` you're using | Always use `model.Todo` (qualified name) when in different packages |

### When Are You Ready for This?

- [x] Completed Approach 2 (struct + methods)
- [ ] Comfortable with Approach 3 (interfaces) — recommended but not required
- [ ] Your single file is getting hard to navigate
- [ ] You want to understand how real Go projects are organized

**See the full detailed guide with code examples:** [`approach-4-packages-modules.md`](./approach-4-packages-modules.md)

---

## 7. Comparison Summary

| Criteria | Global Variable (Approach 1) | Struct + Methods (Approach 2) | Interface + DI (Approach 3) | Multi-File/Packages (Approach 4) |
|---|---|---|---|---|
| **Complexity** | Lowest | Medium | High | Highest |
| **Testability** | Hard | Easy | Easiest | Easiest (per-package tests) |
| **Explicit deps** | No (hidden) | Yes (in struct) | Yes (in interface) | Yes (per package boundary) |
| **Learning value** | Low | High | Very High | Very High (real-world skill) |
| **Your stage** | You're past this | Do this now | After Approach 2 | After Approaches 2 & 3 |
| **Real-world use** | Scripts, CLIs | Most Go apps | Libraries, large apps | **Every real Go project** |

---

## 8. Your Action Plan
|---|---|---|---|
| **Complexity** | Lowest | Medium | Highest |
| **Testability** | Hard | Easy | Easiest |
| **Explicit deps** | No (hidden) | Yes (in struct) | Yes (in interface) |
| **Learning value** | Low | High | High (but advanced) |
| **Your stage** | You're past this | Do this now | Save for later |
| **Real-world use** | Scripts, CLIs | Most Go apps | Libraries, large apps |

---

## 7. Your Action Plan

Follow these steps in order. Do each step, run your program, verify it still works, then move to the next.

### Phase 1: Create the Repository Struct

1. Define `type TodoRepo struct { db *sql.DB }` near your existing `Todo` struct
2. Write `func NewTodoRepo(db *sql.DB) *TodoRepo`
3. Pick ONE function (start with `getTodos`) and convert it to `func (r *TodoRepo) GetTodos(page int)`
4. Change all internal `db.` references to `r.db.`
5. Update the corresponding handler to accept `*TodoRepo` and call `repo.GetTodos()`
6. Update `main()` to create `repo := NewTodoRepo(db)` and pass it
7. **Test it.** Run `go run main.go list` and confirm it works.

### Phase 2: Convert Remaining Functions

8. Repeat the conversion for: `createTask` → `Create`, `updateTask` → `Update`, `toggleTask` → `Toggle`, `deleteTask` → `Delete`
9. Update all handlers to use the new method calls
10. **Test everything.** Run all commands: create, list, update, toggle, delete.

### Phase 3: Clean Up (Optional)

11. Remove old non-method versions of functions (keep only the methods)
12. Make sure program compiles cleanly with `go build`
13. Commit your changes

### After Completing Approaches 2 & 3

You'll be ready for **Approach 4: Multi-File & Multi-Package Architecture** — splitting your code into proper Go packages (`model/`, `repository/`, `handler/`). See [`approach-4-packages-modules.md`](./approach-4-packages-modules.md) for the full guide.

---

## Quick Reference Card

```
FUNCTION → METHOD CONVERSION RULES
═══════════════════════════════════

OLD:  func getTodos(db *sql.DB, page int) ([]Todo, int, error)
NEW:  func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error)

OLD:  db.Query(...)
NEW:  r.db.Query(...)

OLD:  handleCreate(db, args)  →  createTask(db, task)
NEW:  handleCreate(repo, args)  →  repo.Create(task)

OLD:  db := InitDB()  →  handleCreate(db, args)
NEW:  repo := NewTodoRepo(db)  →  handleCreate(repo, args)
```
