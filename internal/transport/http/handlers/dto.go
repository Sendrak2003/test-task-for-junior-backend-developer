package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type RecurrenceDTO struct {
	Type          string   `json:"type"`
	DayInterval   *int     `json:"day_interval,omitempty"`
	MonthDays     []int    `json:"month_days,omitempty"`
	SpecificDates []string `json:"specific_dates,omitempty"`
	EvenOddType   *string  `json:"even_odd_type,omitempty"`
}

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	// **RecurrenceDTO для partial update:
	//   nil    — поле отсутствует в JSON (не изменять)
	//   &nil   — поле передано как null (удалить)
	//   &dto   — поле передано со значением (заменить)
	Recurrence **RecurrenceDTO `json:"recurrence"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Recurrence  *RecurrenceDTO    `json:"recurrence"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Recurrence:  recurrenceDomainToDTO(task.Recurrence),
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func recurrenceDTOToDomain(dto *RecurrenceDTO) *taskdomain.RecurrenceSettings {
	if dto == nil {
		return nil
	}

	return &taskdomain.RecurrenceSettings{
		Type:          taskdomain.RecurrenceType(dto.Type),
		DayInterval:   dto.DayInterval,
		MonthDays:     dto.MonthDays,
		SpecificDates: dto.SpecificDates,
		EvenOddType:   dto.EvenOddType,
	}
}

func recurrenceDomainToDTO(r *taskdomain.RecurrenceSettings) *RecurrenceDTO {
	if r == nil {
		return nil
	}

	return &RecurrenceDTO{
		Type:          string(r.Type),
		DayInterval:   r.DayInterval,
		MonthDays:     r.MonthDays,
		SpecificDates: r.SpecificDates,
		EvenOddType:   r.EvenOddType,
	}
}
