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
