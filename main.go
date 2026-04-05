package main

import (
	//"log"
	"fmt"
	"os"
	"strconv"
	//_ "github.com/mattn/go-sqlite3"
)

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

func handleList(args []string) {
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

	fmt.Println("Listing page:", page)
}

func handleCreate(args []string) {
	if len(args) < 2 {
		fmt.Println("Name is required")
		fmt.Println("Usage: create <task>")
		return
	}

	if len(args) > 2 {
		fmt.Println("Too many arguments for 'create'")
		return
	}

	task := args[1]
	fmt.Println("Created task:", task)
}

func handleDelete(args []string) {
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

	fmt.Println("Deleted task with ID:", id)
}

func handleUpdate(args []string) {
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

	name := args[2]

	if len(args) > 3 {
		fmt.Println("Too many arguments for 'update'")
		return
	}

	fmt.Println("Updated task:", id, "->", name)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("No command provided")
		PrintHelp()
		return
	}

	switch args[0] {

	case "help", "h":
		PrintHelp()

	case "list":
		handleList(args)

	case "create":
		handleCreate(args)

	case "delete":
		handleDelete(args)

	case "update":
		handleUpdate(args)

	case "version", "v":
		fmt.Println("App Version: 1.0.0")

	default:
		fmt.Println("Unknown command")
		PrintHelp()
	}

}
