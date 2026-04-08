package main

import "time"

type Todo struct {
	ID        int
	Task      string
	Done      bool
	CreatedAt time.Time
}

const LIMIT = 10
