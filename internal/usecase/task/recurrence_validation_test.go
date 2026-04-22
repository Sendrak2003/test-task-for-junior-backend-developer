package task

import (
	"errors"
	"testing"

	taskdomain "example.com/taskservice/internal/domain/task"
)

func ptr[T any](v T) *T { return &v }

func TestValidRecurrenceAccepted(t *testing.T) {
	cases := []struct {
		name string
		r    *taskdomain.RecurrenceSettings
	}{
		{"nil recurrence", nil},
		{"daily interval=1", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(1)}},
		{"daily interval=365", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(365)}},
		{"daily interval=7", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(7)}},
		{"monthly single day", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly, MonthDays: []int{1}}},
		{"monthly multiple days", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly, MonthDays: []int{1, 15, 30}}},
		{"specific_dates single", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates, SpecificDates: []string{"2026-01-01"}}},
		{"specific_dates multiple", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates, SpecificDates: []string{"2026-01-01", "2026-12-31"}}},
		{"even_odd_days even", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceEvenOddDays, EvenOddType: ptr("even")}},
		{"even_odd_days odd", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceEvenOddDays, EvenOddType: ptr("odd")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateRecurrence(tc.r); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestInvalidRecurrenceRejected(t *testing.T) {
	cases := []struct {
		name string
		r    *taskdomain.RecurrenceSettings
	}{
		{"empty type", &taskdomain.RecurrenceSettings{Type: ""}},
		{"unknown type", &taskdomain.RecurrenceSettings{Type: "weekly"}},
		{"daily missing day_interval", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily}},
		{"daily day_interval=0", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(0)}},
		{"daily day_interval=366", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(366)}},
		{"daily day_interval=-1", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceDaily, DayInterval: ptr(-1)}},
		{"monthly empty month_days", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly, MonthDays: []int{}}},
		{"monthly nil month_days", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly}},
		{"monthly day=0", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly, MonthDays: []int{0}}},
		{"monthly day=31", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceMonthly, MonthDays: []int{31}}},
		{"specific_dates empty", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates, SpecificDates: []string{}}},
		{"specific_dates nil", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates}},
		{"specific_dates invalid format", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates, SpecificDates: []string{"01-01-2026"}}},
		{"specific_dates not a date", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceSpecificDates, SpecificDates: []string{"not-a-date"}}},
		{"even_odd_days missing even_odd_type", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceEvenOddDays}},
		{"even_odd_days invalid value", &taskdomain.RecurrenceSettings{Type: taskdomain.RecurrenceEvenOddDays, EvenOddType: ptr("both")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRecurrence(tc.r)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidRecurrence) {
				t.Errorf("expected ErrInvalidRecurrence, got: %v", err)
			}
		})
	}
}

func TestRecurrenceDeduplication(t *testing.T) {
	t.Run("monthly deduplicates and sorts month_days", func(t *testing.T) {
		r := &taskdomain.RecurrenceSettings{
			Type:      taskdomain.RecurrenceMonthly,
			MonthDays: []int{15, 1, 15, 5, 1},
		}
		result := normalizeRecurrence(r)
		want := []int{1, 5, 15}
		if len(result.MonthDays) != len(want) {
			t.Fatalf("expected %v, got %v", want, result.MonthDays)
		}
		for i, v := range want {
			if result.MonthDays[i] != v {
				t.Errorf("index %d: expected %d, got %d", i, v, result.MonthDays[i])
			}
		}
	})

	t.Run("specific_dates deduplicates and sorts", func(t *testing.T) {
		r := &taskdomain.RecurrenceSettings{
			Type:          taskdomain.RecurrenceSpecificDates,
			SpecificDates: []string{"2026-05-01", "2026-01-01", "2026-05-01"},
		}
		result := normalizeRecurrence(r)
		want := []string{"2026-01-01", "2026-05-01"}
		if len(result.SpecificDates) != len(want) {
			t.Fatalf("expected %v, got %v", want, result.SpecificDates)
		}
		for i, v := range want {
			if result.SpecificDates[i] != v {
				t.Errorf("index %d: expected %s, got %s", i, v, result.SpecificDates[i])
			}
		}
	})

	t.Run("nil recurrence returns nil", func(t *testing.T) {
		if normalizeRecurrence(nil) != nil {
			t.Error("expected nil")
		}
	})
}
