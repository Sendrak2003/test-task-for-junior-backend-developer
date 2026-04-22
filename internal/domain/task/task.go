package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthly       RecurrenceType = "monthly"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOddDays   RecurrenceType = "even_odd_days"
)

type RecurrenceSettings struct {
	Type          RecurrenceType `json:"type"`
	DayInterval   *int           `json:"day_interval,omitempty"`
	MonthDays     []int          `json:"month_days,omitempty"`
	SpecificDates []string       `json:"specific_dates,omitempty"`
	EvenOddType   *string        `json:"even_odd_type,omitempty"`
}

type Task struct {
	ID          int64               `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Status      Status              `json:"status"`
	Recurrence  *RecurrenceSettings `json:"recurrence"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
