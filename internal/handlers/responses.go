package handlers

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
}

type GetStatusResponse struct {
	Status string `json:"status"`
}

type GetResultResponse struct {
	Result string `json:"result"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
