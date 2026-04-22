package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type mockUsecase struct {
	createFn  func(ctx context.Context, input taskusecase.CreateInput) (*taskdomain.Task, error)
	getByIDFn func(ctx context.Context, id int64) (*taskdomain.Task, error)
	updateFn  func(ctx context.Context, id int64, input taskusecase.UpdateInput) (*taskdomain.Task, error)
	deleteFn  func(ctx context.Context, id int64) error
	listFn    func(ctx context.Context) ([]taskdomain.Task, error)
}

func (m *mockUsecase) Create(ctx context.Context, input taskusecase.CreateInput) (*taskdomain.Task, error) {
	return m.createFn(ctx, input)
}

func (m *mockUsecase) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUsecase) Update(ctx context.Context, id int64, input taskusecase.UpdateInput) (*taskdomain.Task, error) {
	return m.updateFn(ctx, id, input)
}

func (m *mockUsecase) Delete(ctx context.Context, id int64) error {
	return m.deleteFn(ctx, id)
}

func (m *mockUsecase) List(ctx context.Context) ([]taskdomain.Task, error) {
	return m.listFn(ctx)
}

func ptr[T any](v T) *T { return &v }

func fixedTime() time.Time {
	return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
}

func TestGetTaskIncludesRecurrence(t *testing.T) {
	interval := 3
	task := &taskdomain.Task{
		ID:     1,
		Title:  "Daily task",
		Status: taskdomain.StatusNew,
		Recurrence: &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: &interval,
		},
		CreatedAt: fixedTime(),
		UpdatedAt: fixedTime(),
	}

	dto := newTaskDTO(task)

	if dto.Recurrence == nil {
		t.Fatal("expected recurrence in DTO, got nil")
	}
	if dto.Recurrence.Type != "daily" {
		t.Errorf("expected type=daily, got %s", dto.Recurrence.Type)
	}
	if dto.Recurrence.DayInterval == nil || *dto.Recurrence.DayInterval != 3 {
		t.Errorf("expected day_interval=3, got %v", dto.Recurrence.DayInterval)
	}
}

func TestListTasksIncludeRecurrence(t *testing.T) {
	interval := 2
	evenOdd := "even"
	tasks := []taskdomain.Task{
		{
			ID:     1,
			Title:  "Daily task",
			Status: taskdomain.StatusNew,
			Recurrence: &taskdomain.RecurrenceSettings{
				Type:        taskdomain.RecurrenceDaily,
				DayInterval: &interval,
			},
			CreatedAt: fixedTime(),
			UpdatedAt: fixedTime(),
		},
		{
			ID:     2,
			Title:  "Even days task",
			Status: taskdomain.StatusNew,
			Recurrence: &taskdomain.RecurrenceSettings{
				Type:        taskdomain.RecurrenceEvenOddDays,
				EvenOddType: &evenOdd,
			},
			CreatedAt: fixedTime(),
			UpdatedAt: fixedTime(),
		},
		{
			ID:         3,
			Title:      "No recurrence",
			Status:     taskdomain.StatusNew,
			Recurrence: nil,
			CreatedAt:  fixedTime(),
			UpdatedAt:  fixedTime(),
		},
	}

	uc := &mockUsecase{
		listFn: func(_ context.Context) ([]taskdomain.Task, error) {
			return tasks, nil
		},
	}

	handler := NewTaskHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response []taskDTO
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(response))
	}
	if response[0].Recurrence == nil {
		t.Error("task 1: expected recurrence, got nil")
	} else if response[0].Recurrence.Type != "daily" {
		t.Errorf("task 1: expected daily, got %s", response[0].Recurrence.Type)
	}
	if response[1].Recurrence == nil {
		t.Error("task 2: expected recurrence, got nil")
	} else if response[1].Recurrence.Type != "even_odd_days" {
		t.Errorf("task 2: expected even_odd_days, got %s", response[1].Recurrence.Type)
	}
	if response[2].Recurrence != nil {
		t.Errorf("task 3: expected nil recurrence, got %+v", response[2].Recurrence)
	}
}

func TestCreateReturns400OnInvalidRecurrence(t *testing.T) {
	uc := &mockUsecase{
		createFn: func(_ context.Context, input taskusecase.CreateInput) (*taskdomain.Task, error) {
			return nil, taskusecase.ErrInvalidRecurrence
		},
	}

	handler := NewTaskHandler(uc)
	body := `{"title":"Test","recurrence":{"type":"unknown"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] == "" {
		t.Error("expected error message in response body")
	}
}

func TestRecurrenceDTOMapping(t *testing.T) {
	t.Run("nil domain returns nil DTO", func(t *testing.T) {
		if recurrenceDomainToDTO(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("nil DTO returns nil domain", func(t *testing.T) {
		if recurrenceDTOToDomain(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("daily round-trip", func(t *testing.T) {
		domain := &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: ptr(5),
		}
		dto := recurrenceDomainToDTO(domain)
		back := recurrenceDTOToDomain(dto)
		if back.Type != domain.Type {
			t.Errorf("type mismatch: %s != %s", back.Type, domain.Type)
		}
		if *back.DayInterval != *domain.DayInterval {
			t.Errorf("day_interval mismatch: %d != %d", *back.DayInterval, *domain.DayInterval)
		}
	})

	t.Run("monthly round-trip", func(t *testing.T) {
		domain := &taskdomain.RecurrenceSettings{
			Type:      taskdomain.RecurrenceMonthly,
			MonthDays: []int{1, 15, 30},
		}
		dto := recurrenceDomainToDTO(domain)
		back := recurrenceDTOToDomain(dto)
		if len(back.MonthDays) != len(domain.MonthDays) {
			t.Errorf("month_days length mismatch: %d != %d", len(back.MonthDays), len(domain.MonthDays))
		}
	})

	t.Run("specific_dates round-trip", func(t *testing.T) {
		domain := &taskdomain.RecurrenceSettings{
			Type:          taskdomain.RecurrenceSpecificDates,
			SpecificDates: []string{"2026-01-01", "2026-06-15"},
		}
		dto := recurrenceDomainToDTO(domain)
		back := recurrenceDTOToDomain(dto)
		if len(back.SpecificDates) != len(domain.SpecificDates) {
			t.Errorf("specific_dates length mismatch")
		}
	})

	t.Run("even_odd_days round-trip", func(t *testing.T) {
		domain := &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceEvenOddDays,
			EvenOddType: ptr("odd"),
		}
		dto := recurrenceDomainToDTO(domain)
		back := recurrenceDTOToDomain(dto)
		if *back.EvenOddType != "odd" {
			t.Errorf("even_odd_type mismatch: %s", *back.EvenOddType)
		}
	})
}
