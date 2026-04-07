# Approach 2 — Struct with Methods: Cheat Sheet

## The Big Picture

**Before:** Every function takes `db *sql.DB` as first parameter → threading hell
**After:** `db` lives inside a struct → methods use `r.db` → clean signatures

```
BEFORE:  handleCreate(db, args) → createTask(db, task)
AFTER:   repo := NewTodoRepo(db)
         handleCreate(repo, args) → repo.Create(task)
```

---

## Step 1 — Define the Struct

```go
type TodoRepo struct {
    db *sql.DB
}
```

That's it. One field holding the DB connection.

---

## Step 2 — Write the Constructor

```go
func NewTodoRepo(db *sql.DB) *TodoRepo {
    return &TodoRepo{db: db}
}
```

**Why?** Standard Go convention. Lets you add validation/extra fields later in one place.

---

## Step 3 — Convert Functions to Methods (THE KEY STEP)

### The Transformation Rules

| What | Before | After |
|------|--------|-------|
| **Signature** | `func getTodos(db *sql.DB, page int)` | `func (r *TodoRepo) GetTodos(page int)` |
| **DB access** | `db.Query(...)` | `r.db.Query(...)` |
| **DB access** | `db.Exec(...)` | `r.db.Exec(...)` |
| **DB access** | `db.QueryRow(...)` | `r.db.QueryRow(...)` |

### Full Example: getTodos → GetTodos

```go
// BEFORE
func getTodos(db *sql.DB, page int) ([]Todo, int, error) {
    offset := (page - 1) * LIMIT
    rows, err := db.Query(queryAllSQL, LIMIT, offset)
    // ...
}

// AFTER
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error) {
    offset := (page - 1) * LIMIT
    rows, err := r.db.Query(queryAllSQL, LIMIT, offset)
    // ...
}
```

### All 5 Conversions at a Glance

| Old Function | New Method |
|-------------|------------|
| `func createTask(db *sql.DB, task string)` | `func (r *TodoRepo) Create(task string)` |
| `func getTodos(db *sql.DB, page int)` | `func (r *TodoRepo) GetTodos(page int)` |
| `func updateTask(db *sql.DB, id int, task string)` | `func (r *TodoRepo) Update(id int, task string)` |
| `func toggleTask(db *sql.DB, id int)` | `func (r *TodoRepo) Toggle(id int)` |
| `func deleteTask(db *sql.DB, id int)` | `func (r *TodoRepo) Delete(id int)` |

**Note:** Uppercase first letter = exported (public). Idiomatic Go.

---

## Step 4 — Update Handlers

```go
// BEFORE
func handleCreate(db *sql.DB, args []string) {
    todo, err := createTask(db, task)
}

// AFTER
func handleCreate(repo *TodoRepo, args []string) {
    todo, err := repo.Create(task)
}
```

**Pattern:** Replace `db *sql.DB` with `repo *TodoRepo`, then `repo.MethodName(...)`.

---

## Step 5 — Update main()

```go
func main() {
    db := InitDB()
    defer CloseDB(db)

    repo := NewTodoRepo(db)

    switch args[0] {
    case "create":
        handleCreate(repo, args)
    case "list":
        handleList(repo, args)
    }
}
```

---

## Final Skeleton

```go
package main

type Todo struct { /* ... */ }

type TodoRepo struct { db *sql.DB }

func NewTodoRepo(db *sql.DB) *TodoRepo { return &TodoRepo{db: db} }

func (r *TodoRepo) Create(task string) (Todo, error)       { /* r.db.Exec(...) */ }
func (r *TodoRepo) GetTodos(page int) ([]Todo, int, error)  { /* r.db.Query(...) */ }
func (r *TodoRepo) Update(id int, task string) (Todo, error){ /* r.db.Exec(...) */ }
func (r *TodoRepo) Toggle(id int) (Todo, error)             { /* r.db.Exec(...) */ }
func (r *TodoRepo) Delete(id int) (int64, error)            { /* r.db.Exec(...) */ }

func handleList(repo *TodoRepo, args []string)   { repo.GetTodos(...) }
func handleCreate(repo *TodoRepo, args []string) { repo.Create(...) }
func handleUpdate(repo *TodoRepo, args []string) { repo.Update(...) }
func handleToggle(repo *TodoRepo, args []string) { repo.Toggle(...) }
func handleDelete(repo *TodoRepo, args []string) { repo.Delete(...) }

func main() {
    db := InitDB()
    defer CloseDB(db)
    repo := NewTodoRepo(db)
    // route commands using repo
}
```

---

## Why Is This Better Than What We Have Now?

### Honest answer: yes, you're still passing something through layers

```
CURRENT APPROACH:                    APPROACH 2:
handleCreate(db, args)      →        handleCreate(repo, args)
       ↓                                  ↓
createTask(db, task)         →        repo.Create(task)
                                     ↑
                                db is INSIDE repo
```

Both approaches pass a dependency. So what's actually different?

### The real difference: what happens when you add a 2nd dependency?

Imagine tomorrow you need a **logger**. Or a **config**. Or a **cache**.

| Scenario | Current Approach (passing raw `db`) | Approach 2 (passing `repo`) |
|----------|--------------------------------------|-----------------------------|
| **Add a logger** | Change **every function signature**: `func createTask(db, logger, task)`, `func getTodos(db, logger, page)`, etc. All 5 data functions + all 5 handlers = **10+ signatures rewritten** | Add **one field** to the struct: `type TodoRepo struct { db *sql.DB; logger *log.Logger }`. **Zero** function signatures change. |
| **Add a cache** | Same pain again — every signature grows | Add one more field. Done. Caller still just passes `repo`. |
| **Caller knows about?** | Caller must know about `db`, `logger`, `cache`, and whatever else you add later | Caller only knows about `repo`. Doesn't care what's inside. |

### One-line answer if someone asks "why are you doing this?"

> *"I'm wrapping `db` inside a `TodoRepo` struct so all database logic is self-contained. If I ever need to add a logger, config, or cache — I just add a field to the struct. I don't have to rewrite every single function signature."*

### Summary of Benefits

| Benefit | One-Liner |
|---------|-----------|
| **Explicit deps** | Function signatures show what they need (`repo`), not implementation details (`db`) |
| **Self-contained** | All DB logic lives in `TodoRepo` — one place to look |
| **Testable** | Pass a test DB via `NewTodoRepo(testDB)` — no global state |
| **Scalable** | New dependency = one field added, zero signatures changed |
| **Encapsulated** | Callers use `repo.Create(...)` — they don't know about SQL or `*sql.DB` |
| **Future-proof** | Adding logger/config/cache doesn't ripple through your entire codebase |

---

## Common Mistakes to Avoid

1. **Forgetting to change `db.` to `r.db.` inside the method** → compilation error or nil pointer
2. **Using value receiver `(r TodoRepo)` instead of pointer `(r *TodoRepo)` → can't access `r.db` properly on copies
3. **Not updating all callers** → old functions still expect `db *sql.DB`
4. **Leaving old functions around** → confusion about which version to use
