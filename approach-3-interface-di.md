# Approach 3 — Interface + Dependency Injection (Advanced)

## The Big Idea

**Approach 2** gives you a clean struct with methods. **Approach 3** goes one step further: handlers depend on an **interface** (a contract), not the concrete struct. This means you can swap in **fake implementations for testing** without touching a real database.

```
Approach 2:  handleCreate(repo *TodoRepo, args)    ← depends on concrete struct
Approach 3:  handleCreate(svc TodoRepository, args) ← depends on interface (ANY implementation works)
```

---

## Architecture Diagram

```
                +---------------------+
                |   TodoRepository    |  ← INTERFACE (just a contract)
                |  - Create()         |     No code, only method signatures
                |  - GetTodos()       |
                |  - Update()         |
                |  - Toggle()         |
                |  - Delete()         |
                +---------+-----------+
                          | implements implicitly
          +---------------+---------------+
          v                               v
   +--------------+               +--------------+
   |   TodoRepo   |               | MockTodoRepo |
   |(real SQLite) |               |(fake for test)|
   +--------------+               +--------------+
```

---

## The 4 Steps

### Step 1 — Define the Interface

```go
type TodoRepository interface {
    Create(task string) (Todo, error)
    GetTodos(page int) ([]Todo, int, error)
    Update(id int, task string) (Todo, error)
    Toggle(id int) (Todo, error)
    Delete(id int) (int64, error)
}
```

**Key points:**
- An interface is just a list of method signatures — no implementation
- It defines a **contract**: "anything with these methods is a TodoRepository"
- Name it after the behavior (`TodoRepository`), not the implementation

### Step 2 — Concrete Implementation (Same as Approach 2)

```go
type TodoRepo struct {
    db *sql.DB
}

func NewTodoRepo(db *sql.DB) *TodoRepo {
    return &TodoRepo{db: db}
}

func (r *TodoRepo) Create(task string) (Todo, error)       { /* r.db.Exec(...) */ }
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error)  { /* r.db.Query(...) */ }
func (r *TodoRepo) Update(id int, task string) (Todo, error){ /* r.db.Exec(...) */ }
func (r *TodoRepo) Toggle(id int) (Todo, error)             { /* r.db.Exec(...) */ }
func (r *TodoRepo) Delete(id int) (int64, error)            { /* r.db.Exec(...) */ }
```

**Go's magic:** `TodoRepo` **automatically satisfies** `TodoRepository` because it has all the required methods. No `implements` keyword needed. This is called **structural typing**.

### Step 3 — Handlers Accept the Interface

```go
// BEFORE (Approach 2)
func handleCreate(repo *TodoRepo, args []string) {
    todo, err := repo.Create(task)
}

// AFTER (Approach 3)
func handleCreate(svc TodoRepository, args []string) {
    todo, err := svc.Create(task)  // works with TodoRepo OR MockTodoRepo OR anything else
}
```

**The critical change:** Parameter type is now `TodoRepository` (interface), not `*TodoRepo` (concrete struct).

### Step 4 — Wire It Up in main()

```go
func main() {
    db := InitDB()
    defer CloseDB(db)

    repo := NewTodoRepo(db)  // concrete implementation

    // Pass to handlers — Go automatically treats repo as TodoRepository
    switch args[0] {
    case "create":
        handleCreate(repo, args)  // *TodoRepo → TodoRepository (implicit)
    case "list":
        handleList(repo, args)
    }
}
```

---

## Why Interfaces? The Real Power: Mocking

### Without Interface (Approach 2)
To test `handleCreate`, you need a real SQLite database. Tests are slow and fragile.

### With Interface (Approach 3)
You can create a fake implementation that returns hardcoded data:

```go
// A fake implementation for testing — no database needed!
type MockTodoRepo struct {
    todos []Todo
}

func (m *MockTodoRepo) Create(task string) (Todo, error) {
    todo := Todo{ID: 999, Task: task, Done: false}
    m.todos = append(m.todos, todo)
    return todo, nil
}

func (m *MockTodoRepo) GetTodos(page int) ([]Todo, int, error) {
    return m.todos, len(m.todos), nil
}

func (m *MockTodoRepo) Update(id int, task string) (Todo, error) {
    for i := range m.todos {
        if m.todos[i].ID == id {
            m.todos[i].Task = task
            return m.todos[i], nil
        }
    }
    return Todo{}, fmt.Errorf("not found")
}

func (m *MockTodoRepo) Toggle(id int) (Todo, error) {
    for i := range m.todos {
        if m.todos[i].ID == id {
            m.todos[i].Done = !m.todos[i].Done
            return m.todos[i], nil
        }
    }
    return Todo{}, fmt.Errorf("not found")
}

func (m *MockTodoRepo) Delete(id int) (int64, error) {
    for i, t := range m.todos {
        if t.ID == id {
            m.todos = append(m.todos[:i], m.todos[i+1:]...)
            return 1, nil
        }
    }
    return 0, fmt.Errorf("not found")
}
```

### Using the Mock in Tests

```go
func TestHandleCreate(t *testing.T) {
    mock := &MockTodoRepo{}

    // This works because MockTodoRepo satisfies TodoRepository!
    handleCreate(mock, []string{"create", "buy milk"})

    if len(mock.todos) != 1 {
        t.Fatalf("expected 1 todo, got %d", len(mock.todos))
    }
    if mock.todos[0].Task != "buy milk" {
        t.Errorf("expected task 'buy milk', got '%s'", mock.todos[0].Task)
    }
}
```

**No SQLite, no file I/O, no setup/teardown. Tests run in milliseconds.**

---

## Key Concepts Cheat Sheet

### Implicit Satisfaction (Structural Typing)

```go
// In Java/C#:  class TodoRepo implements TodoRepository { ... }
// In Go:        Just define the methods. That's it. No extra syntax.

type TodoRepository interface { Create() }

type TodoRepo struct { db *sql.DB }
func (r *TodoRepo) Create(task string) (Todo, error) { ... }

// ✅ TodoRepo IS-A TodoRepository automatically
// Go checks this at compile time
```

### Interface Values Under the Hood

```go
var repo TodoRepository = NewTodoRepo(db)
// repo is a pair of: (concrete_value, concrete_type)
// When you call repo.Create(), Go uses the concrete type's method
```

### The Empty Interface `interface{}` / `any`

```go
// Accepts ANY value — use sparingly
func PrintAnything(v any) {
    fmt.Println(v)
}
```

### Best Practices

| Do | Don't |
|----|-------|
| Define interfaces where you **use** them (e.g., in handler params) | Define interfaces next to the concrete type ("interface on the producer") |
| Keep interfaces **small** (1–3 methods) | Make god interfaces with 10+ methods |
| Return concrete types from functions | Return interfaces from functions (unless necessary) |

---

## When to Use Each Approach

| Situation | Use Approach 2 | Use Approach 3 |
|-----------|---------------|----------------|
| Learning Go basics | ✅ Start here | Overkill |
| Small CLI/script | ✅ Fine as-is | Unnecessary |
| Writing unit tests | Works but needs real DB | ✅ Perfect — use mocks |
| Multiple DB backends (SQLite + Postgres) | Need separate structs anyway | ✅ Clean swap via interface |
| Building a library/API for others | Users want flexibility | ✅ Let users inject their own impl |
| Only one DB, no tests yet | ✅ Don't over-engineer | YAGNI |

---

## Golden Rule

> **"Accept interfaces, return structs."**
>
> Your functions should **accept** interface types (for flexibility) but **return** concrete types (for simplicity). And most importantly:
>
> **"Prefer concrete types until the abstraction proves necessary."**
>
> Don't add an interface just because you might need it someday. Add it when you actually write your first test or second implementation.

---

## Full Code Skeleton

```go
package main

// --- MODEL ---
type Todo struct {
    ID        int
    Task      string
    Done      bool
    CreatedAt time.Time
}

// --- INTERFACE ---
type TodoRepository interface {
    Create(task string) (Todo, error)
    GetTodos(page int) ([]Todo, int, error)
    Update(id int, task string) (Todo, error)
    Toggle(id int) (Todo, error)
    Delete(id int) (int64, error)
}

// --- CONCRETE IMPLEMENTATION (Real DB) ---
type TodoRepo struct {
    db *sql.DB
}

func NewTodoRepo(db *sql.DB) *TodoRepo {
    return &TodoRepo{db: db}
}

func (r *TodoRepo) Create(task string) (Todo, error)       { /* r.db */ }
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error)  { /* r.db */ }
func (r *TodoRepo) Update(id int, task string) (Todo, error){ /* r.db */ }
func (r *TodoRepo) Toggle(id int) (Todo, error)             { /* r.db */ }
func (r *TodoRepo) Delete(id int) (int64, error)            { /* r.db */ }

// --- MOCK IMPLEMENTATION (For Testing) ---
type MockTodoRepo struct {
    todos []Todo
}

func (m *MockTodoRepo) Create(task string) (Todo, error)     { /* in-memory */ }
func (m *MockTodoRepo) GetTodos(page int) ([]Todo, int, error){ /* in-memory */ }
func (m *MockTodoRepo) Update(id int, task string) (Todo, error){ /* in-memory */ }
func (m *MockTodoRepo) Toggle(id int) (Todo, error)           { /* in-memory */ }
func (m *MockTodoRepo) Delete(id int) (int64, error)          { /* in-memory */ }

// --- HANDLERS (depend on INTERFACE, not concrete) ---
func handleList(svc TodoRepository, args []string)   { svc.GetTodos(...) }
func handleCreate(svc TodoRepository, args []string) { svc.Create(...) }
func handleUpdate(svc TodoRepository, args []string) { svc.Update(...) }
func handleToggle(svc TodoRepository, args []string) { svc.Toggle(...) }
func handleDelete(svc TodoRepository, args []string) { svc.Delete(...) }

// --- MAIN ---
func main() {
    db := InitDB()
    defer CloseDB(db)

    repo := NewTodoRepo(db)  // concrete

    switch args[0] {
    case "create": handleCreate(repo, args)   // passed as TodoRepository
    case "list":   handleList(repo, args)
    }
}
```
