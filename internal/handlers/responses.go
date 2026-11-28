package handlers

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
}

type GetStatusResponse struct {
	Status string `json:"status"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
