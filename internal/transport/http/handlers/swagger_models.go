package handlers

// CreateTaskRequest описывает тело запроса для создания задачи.
// swagger:model
type CreateTaskRequest struct {
	Title       string         `json:"title" example:"Обход пациентов"`
	Description string         `json:"description" example:"Провести утренний обход пациентов"`
	Status      string         `json:"status" enums:"new,in_progress,done" example:"new"`
	Recurrence  *RecurrenceDTO `json:"recurrence"`
}

// UpdateTaskRequest описывает тело запроса для обновления задачи.
// swagger:model
type UpdateTaskRequest struct {
	Title       string         `json:"title" example:"Обход пациентов"`
	Description string         `json:"description" example:"Провести утренний обход пациентов"`
	Status      string         `json:"status" enums:"new,in_progress,done" example:"in_progress"`
	Recurrence  *RecurrenceDTO `json:"recurrence"`
}

// ErrorResponse описывает тело ответа при ошибке.
// swagger:model
type ErrorResponse struct {
	Error string `json:"error" example:"task not found"`
}
