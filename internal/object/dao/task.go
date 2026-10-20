package dao

type Task struct {
	ID        int
	Title     string
	Completed bool
	Assignee  string
}

type TaskFilter struct {
	Assignee  *string
	Completed *bool
}
