package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		Recurrence:  normalized.Recurrence,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, model)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}

	switch {
	case normalized.Recurrence == nil:
		// поле не передано — сохраняем существующее
		model.Recurrence = existing.Recurrence
	case *normalized.Recurrence == nil:
		// передан null — удаляем периодичность
		model.Recurrence = nil
	default:
		// передано значение — заменяем
		model.Recurrence = *normalized.Recurrence
	}

	return s.repo.Update(ctx, model)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if err := validateRecurrence(input.Recurrence); err != nil {
		return CreateInput{}, err
	}

	input.Recurrence = normalizeRecurrence(input.Recurrence)

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.Recurrence != nil && *input.Recurrence != nil {
		if err := validateRecurrence(*input.Recurrence); err != nil {
			return UpdateInput{}, err
		}

		normalized := normalizeRecurrence(*input.Recurrence)
		input.Recurrence = &normalized
	}

	return input, nil
}

func validateRecurrence(r *taskdomain.RecurrenceSettings) error {
	if r == nil {
		return nil
	}

	if r.Type == "" {
		return fmt.Errorf("%w: type is required", ErrInvalidRecurrence)
	}

	switch r.Type {
	case taskdomain.RecurrenceDaily:
		if r.DayInterval == nil || *r.DayInterval == 0 {
			return fmt.Errorf("%w: day_interval is required for daily type", ErrInvalidRecurrence)
		}

		if *r.DayInterval < 1 || *r.DayInterval > 365 {
			return fmt.Errorf("%w: day_interval must be between 1 and 365", ErrInvalidRecurrence)
		}

	case taskdomain.RecurrenceMonthly:
		if len(r.MonthDays) == 0 {
			return fmt.Errorf("%w: month_days is required for monthly type", ErrInvalidRecurrence)
		}

		for _, d := range r.MonthDays {
			if d < 1 || d > 30 {
				return fmt.Errorf("%w: month_days values must be between 1 and 30", ErrInvalidRecurrence)
			}
		}

	case taskdomain.RecurrenceSpecificDates:
		if len(r.SpecificDates) == 0 {
			return fmt.Errorf("%w: specific_dates is required for specific_dates type", ErrInvalidRecurrence)
		}

		for _, d := range r.SpecificDates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("%w: specific_dates contains invalid date", ErrInvalidRecurrence)
			}
		}

	case taskdomain.RecurrenceEvenOddDays:
		if r.EvenOddType == nil {
			return fmt.Errorf("%w: even_odd_type is required for even_odd_days type", ErrInvalidRecurrence)
		}

		if *r.EvenOddType != "even" && *r.EvenOddType != "odd" {
			return fmt.Errorf("%w: even_odd_type must be 'even' or 'odd'", ErrInvalidRecurrence)
		}

	default:
		return fmt.Errorf("%w: unknown recurrence type", ErrInvalidRecurrence)
	}

	return nil
}

func normalizeRecurrence(r *taskdomain.RecurrenceSettings) *taskdomain.RecurrenceSettings {
	if r == nil {
		return nil
	}

	result := *r

	if len(r.MonthDays) > 0 {
		result.MonthDays = deduplicateInts(r.MonthDays)
	}

	if len(r.SpecificDates) > 0 {
		result.SpecificDates = deduplicateStrings(r.SpecificDates)
	}

	return &result
}

func deduplicateInts(s []int) []int {
	seen := make(map[int]struct{}, len(s))
	result := make([]int, 0, len(s))

	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	sort.Ints(result)

	return result
}

func deduplicateStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	result := make([]string, 0, len(s))

	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	sort.Strings(result)

	return result
}
