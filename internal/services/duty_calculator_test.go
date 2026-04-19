package services

import (
	"testing"
	"time"
)

func TestISOWeeksBetween(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		want  int
	}{
		{"same year", "2025-11-04", "2025-12-15", 6},
		{"cross year 2025→2026", "2025-11-04", "2026-01-12", 10},
		{"cross year 2026→2027 (2026 has ISO week 53)", "2026-09-01", "2027-01-11", 19},
		{"cross year 2020→2021 (2020 has ISO week 53)", "2020-09-01", "2021-02-01", 22},
		{"cross multiple years 2025→2027", "2025-09-01", "2027-03-01", 78},
		{"same week", "2025-11-03", "2025-11-07", 0},
		{"exactly one week", "2025-11-03", "2025-11-10", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, _ := time.Parse("2006-01-02", tc.start)
			end, _ := time.Parse("2006-01-02", tc.end)
			got := isoWeeksBetween(start, end)
			if got != tc.want {
				t.Errorf("isoWeeksBetween(%s, %s) = %d, want %d", tc.start, tc.end, got, tc.want)
			}
		})
	}
}
