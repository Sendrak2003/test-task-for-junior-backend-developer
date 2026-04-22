package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type mockRepo struct {
	tasks  map[int64]*taskdomain.Task
	nextID int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{tasks: make(map[int64]*taskdomain.Task), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	task.ID = m.nextID
	m.nextID++
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockRepo) GetByID(_ context.Context, id int64) (*taskdomain.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, taskdomain.ErrNotFound
	}
	return t, nil
}

func (m *mockRepo) Update(_ context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	if _, ok := m.tasks[task.ID]; !ok {
		return nil, taskdomain.ErrNotFound
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockRepo) Delete(_ context.Context, id int64) error {
	if _, ok := m.tasks[id]; !ok {
		return taskdomain.ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *mockRepo) List(_ context.Context) ([]taskdomain.Task, error) {
	result := make([]taskdomain.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		result = append(result, *t)
	}
	return result, nil
}

func fixedNow() time.Time {
	return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
}

func newTestService() *Service {
	svc := NewService(newMockRepo())
	svc.now = fixedNow
	return svc
}

func TestCreateTaskWithoutRecurrence(t *testing.T) {
	svc := newTestService()

	task, err := svc.Create(context.Background(), CreateInput{Title: "Test task"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Recurrence != nil {
		t.Errorf("expected nil recurrence, got %+v", task.Recurrence)
	}
}

func TestCreateTaskWithRecurrence(t *testing.T) {
	svc := newTestService()

	interval := 3
	task, err := svc.Create(context.Background(), CreateInput{
		Title: "Daily task",
		Recurrence: &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: &interval,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Recurrence == nil {
		t.Fatal("expected recurrence, got nil")
	}
	if task.Recurrence.Type != taskdomain.RecurrenceDaily {
		t.Errorf("expected daily, got %s", task.Recurrence.Type)
	}
	if *task.Recurrence.DayInterval != 3 {
		t.Errorf("expected day_interval=3, got %d", *task.Recurrence.DayInterval)
	}
}

func TestUpdateRecurrenceWithNull(t *testing.T) {
	svc := newTestService()

	interval := 1
	created, _ := svc.Create(context.Background(), CreateInput{
		Title: "Task with recurrence",
		Recurrence: &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: &interval,
		},
	})

	var nilRecurrence *taskdomain.RecurrenceSettings
	updated, err := svc.Update(context.Background(), created.ID, UpdateInput{
		Title:      "Task with recurrence",
		Status:     taskdomain.StatusNew,
		Recurrence: &nilRecurrence,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Recurrence != nil {
		t.Errorf("expected nil recurrence after null update, got %+v", updated.Recurrence)
	}
}

func TestUpdateRecurrenceWithValue(t *testing.T) {
	svc := newTestService()

	interval := 1
	created, _ := svc.Create(context.Background(), CreateInput{
		Title: "Task",
		Recurrence: &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: &interval,
		},
	})

	newRecurrence := &taskdomain.RecurrenceSettings{
		Type:      taskdomain.RecurrenceMonthly,
		MonthDays: []int{1, 15},
	}
	updated, err := svc.Update(context.Background(), created.ID, UpdateInput{
		Title:      "Task",
		Status:     taskdomain.StatusNew,
		Recurrence: &newRecurrence,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Recurrence == nil {
		t.Fatal("expected recurrence, got nil")
	}
	if updated.Recurrence.Type != taskdomain.RecurrenceMonthly {
		t.Errorf("expected monthly, got %s", updated.Recurrence.Type)
	}
}

func TestUpdateWithoutRecurrencePreservesSettings(t *testing.T) {
	svc := newTestService()

	interval := 7
	created, _ := svc.Create(context.Background(), CreateInput{
		Title: "Task",
		Recurrence: &taskdomain.RecurrenceSettings{
			Type:        taskdomain.RecurrenceDaily,
			DayInterval: &interval,
		},
	})

	updated, err := svc.Update(context.Background(), created.ID, UpdateInput{
		Title:      "Task updated",
		Status:     taskdomain.StatusInProgress,
		Recurrence: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Recurrence == nil {
		t.Fatal("expected recurrence to be preserved, got nil")
	}
	if updated.Recurrence.Type != taskdomain.RecurrenceDaily {
		t.Errorf("expected daily, got %s", updated.Recurrence.Type)
	}
	if *updated.Recurrence.DayInterval != 7 {
		t.Errorf("expected day_interval=7, got %d", *updated.Recurrence.DayInterval)
	}
}

func TestEvenOddTypeValidValues(t *testing.T) {
	for _, v := range []string{"even", "odd"} {
		t.Run(v, func(t *testing.T) {
			r := &taskdomain.RecurrenceSettings{
				Type:        taskdomain.RecurrenceEvenOddDays,
				EvenOddType: &v,
			}
			if err := validateRecurrence(r); err != nil {
				t.Errorf("expected no error for even_odd_type=%s, got: %v", v, err)
			}
		})
	}
}

func TestValidationErrorIsErrInvalidRecurrence(t *testing.T) {
	svc := newTestService()

	_, err := svc.Create(context.Background(), CreateInput{
		Title: "Bad task",
		Recurrence: &taskdomain.RecurrenceSettings{
			Type: "unknown_type",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidRecurrence) {
		t.Errorf("expected ErrInvalidRecurrence, got: %v", err)
	}
}
