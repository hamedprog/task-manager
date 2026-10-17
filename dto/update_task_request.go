package dto

type UpdateTaskRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}
