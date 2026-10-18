package dto

type TaskResponse struct {
	ID        int    `json:"id"`
	Title     string `json:"title" binding:"required"`
	Completed bool   `json:"completed"`
}
