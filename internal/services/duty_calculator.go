package services

import (
	"fmt"
	"math/rand"
	"teacher-wails/internal/models"
	"teacher-wails/internal/utils"
	"time"
)

// DutyCalculator computes daily duty and weekly lunch assignments.
type DutyCalculator struct {
	dh *DataHandler
	sm *StudentManager
}

// NewDutyCalculator creates a DutyCalculator.
func NewDutyCalculator(dh *DataHandler, sm *StudentManager) *DutyCalculator {
	return &DutyCalculator{dh: dh, sm: sm}
}

// GetTodayDuty returns today's duty students and lunch assignments.
func (dc *DutyCalculator) GetTodayDuty() (models.TodayDutyResult, error) {
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	result := models.TodayDutyResult{
		Date:             utils.FormatDate(today),
		DisplayDate:      utils.FormatDisplayDate(today),
		IsWorkday:        false,
		DutyStudents:     []models.Student{},
		LunchAssignments: []models.LunchAssignment{},
	}

	settings, err := dc.dh.GetSettings()
	if err != nil {
		return result, err
	}

	if settings.SemesterStartDate == "" {
		return result, nil
	}

	startDate, err := utils.ParseDate(settings.SemesterStartDate)
	if err != nil {
		return result, nil
	}

	holidays, err := dc.dh.GetHolidays()
	if err != nil {
		return result, err
	}

	if !utils.IsWorkday(today, holidays) {
		return result, nil
	}
	result.IsWorkday = true

	workdays := utils.CountWorkdays(startDate, today, holidays)

	dutyPool, err := dc.sm.GetEnabledStudents("duty")
	if err != nil {
		return result, err
	}
	lunchPool, err := dc.sm.GetEnabledStudents("lunch")
	if err != nil {
		return result, err
	}

	// Daily duty rotation
	dutyGroupSize := settings.DutyGroupSize
	if dutyGroupSize <= 0 {
		dutyGroupSize = 2
	}
	if len(dutyPool) > 0 {
		result.DutyStudents = calculateGroupRotation(dutyPool, workdays, dutyGroupSize, "daily", today, startDate, settings.DutyStartNumber)
	}

	// Weekly lunch rotation
	lunchGroupSize := settings.LunchGroupSize
	if lunchGroupSize <= 0 {
		lunchGroupSize = 5
	}

	// ISO week-based rotation count
	startYear, startWeek := startDate.ISOWeek()
	currentYear, currentWeek := today.ISOWeek()
	rotationCount := (currentYear-startYear)*52 + (currentWeek - startWeek)

	if len(lunchPool) > 0 {
		lunchStudents := calculateGroupRotation(lunchPool, workdays, lunchGroupSize, "weekly", today, startDate, settings.LunchStartNumber)
		result.LunchAssignments = assignMealBuckets(lunchStudents, settings.MealBuckets, rotationCount)
	}

	return result, nil
}

// GetDutyForDate returns duty info for a specific date (used by schedule export).
func (dc *DutyCalculator) GetDutyForDate(date time.Time) ([]models.Student, []models.LunchAssignment, error) {
	settings, err := dc.dh.GetSettings()
	if err != nil {
		return nil, nil, err
	}
	if settings.SemesterStartDate == "" {
		return nil, nil, nil
	}
	startDate, err := utils.ParseDate(settings.SemesterStartDate)
	if err != nil {
		return nil, nil, nil
	}
	holidays, err := dc.dh.GetHolidays()
	if err != nil {
		return nil, nil, err
	}
	if !utils.IsWorkday(date, holidays) {
		return nil, nil, nil
	}

	workdays := utils.CountWorkdays(startDate, date, holidays)

	dutyPool, _ := dc.sm.GetEnabledStudents("duty")
	lunchPool, _ := dc.sm.GetEnabledStudents("lunch")

	dutyGroupSize := settings.DutyGroupSize
	if dutyGroupSize <= 0 {
		dutyGroupSize = 2
	}
	lunchGroupSize := settings.LunchGroupSize
	if lunchGroupSize <= 0 {
		lunchGroupSize = 5
	}

	var dutyStudents []models.Student
	if len(dutyPool) > 0 {
		dutyStudents = calculateGroupRotation(dutyPool, workdays, dutyGroupSize, "daily", date, startDate, settings.DutyStartNumber)
	}

	startYear, startWeek := startDate.ISOWeek()
	currentYear, currentWeek := date.ISOWeek()
	rotationCount := (currentYear-startYear)*52 + (currentWeek - startWeek)

	var lunchAssignments []models.LunchAssignment
	if len(lunchPool) > 0 {
		lunchStudents := calculateGroupRotation(lunchPool, workdays, lunchGroupSize, "weekly", date, startDate, settings.LunchStartNumber)
		lunchAssignments = assignMealBuckets(lunchStudents, settings.MealBuckets, rotationCount)
	}

	return dutyStudents, lunchAssignments, nil
}

func calculateGroupRotation(students []models.Student, workdays int, groupSize int, rotationType string, currentDate, startDate time.Time, startNumber int) []models.Student {
	if len(students) == 0 || groupSize <= 0 {
		return []models.Student{}
	}

	var rotationCount int
	switch rotationType {
	case "daily":
		rotationCount = workdays
	case "weekly":
		startYear, startWeek := startDate.ISOWeek()
		currentYear, currentWeek := currentDate.ISOWeek()
		rotationCount = (currentYear-startYear)*52 + (currentWeek - startWeek)
	default:
		rotationCount = 0
	}

	// Find the index of the start seat number
	startSeatIndex := 0
	for i, s := range students {
		if s.SeatNumber == startNumber {
			startSeatIndex = i
			break
		}
	}

	// Calculate starting index with modulo wrap
	n := len(students)
	startIndex := ((startSeatIndex + rotationCount*groupSize) % n + n) % n

	group := make([]models.Student, 0, groupSize)
	for i := 0; i < groupSize; i++ {
		idx := (startIndex + i) % n
		group = append(group, students[idx])
	}
	return group
}

func assignMealBuckets(students []models.Student, mealBuckets []string, rotationCount int) []models.LunchAssignment {
	if len(students) == 0 {
		return []models.LunchAssignment{}
	}

	// Build available bucket list
	needed := len(students)
	available := make([]string, 0, needed)
	if len(mealBuckets) >= needed {
		available = append(available, mealBuckets[:needed]...)
	} else {
		available = append(available, mealBuckets...)
		for i := len(mealBuckets); i < needed; i++ {
			available = append(available, fmt.Sprintf("任務%d", i-len(mealBuckets)+1))
		}
	}

	// Seeded Fisher-Yates shuffle
	rng := rand.New(rand.NewSource(int64(rotationCount)))
	for i := len(available) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		available[i], available[j] = available[j], available[i]
	}

	assignments := make([]models.LunchAssignment, len(students))
	for i, s := range students {
		assignments[i] = models.LunchAssignment{
			Student: s,
			Bucket:  available[i],
		}
	}
	return assignments
}
