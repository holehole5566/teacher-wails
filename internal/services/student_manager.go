package services

import (
	"fmt"
	"sort"
	"teacher-wails/internal/models"
)

// StudentManager handles student CRUD operations.
type StudentManager struct {
	dh *DataHandler
}

// NewStudentManager creates a StudentManager backed by the given DataHandler.
func NewStudentManager(dh *DataHandler) *StudentManager {
	return &StudentManager{dh: dh}
}

// GetAllStudents returns all students sorted by seat number.
func (sm *StudentManager) GetAllStudents() ([]models.Student, error) {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return nil, err
	}
	sort.Slice(students, func(i, j int) bool {
		return students[i].SeatNumber < students[j].SeatNumber
	})
	return students, nil
}

// GetEnabledStudents returns students enabled for the given duty type, sorted by seat number.
// dutyType: "duty", "lunch", or "all".
func (sm *StudentManager) GetEnabledStudents(dutyType string) ([]models.Student, error) {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return nil, err
	}

	var enabled []models.Student
	for _, s := range students {
		switch dutyType {
		case "duty":
			if s.DutyEnabled {
				enabled = append(enabled, s)
			}
		case "lunch":
			if s.LunchEnabled {
				enabled = append(enabled, s)
			}
		default:
			if s.DutyEnabled || s.LunchEnabled {
				enabled = append(enabled, s)
			}
		}
	}

	sort.Slice(enabled, func(i, j int) bool {
		return enabled[i].SeatNumber < enabled[j].SeatNumber
	})
	return enabled, nil
}

// AddStudent adds a new student. Returns an error if the seat number already exists.
func (sm *StudentManager) AddStudent(seatNumber int, name string) error {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return err
	}
	for _, s := range students {
		if s.SeatNumber == seatNumber {
			return fmt.Errorf("座號 %d 已存在", seatNumber)
		}
	}
	students = append(students, models.Student{
		SeatNumber:   seatNumber,
		Name:         name,
		DutyEnabled:  true,
		LunchEnabled: true,
	})
	return sm.dh.SaveStudents(students)
}

// UpdateStudent updates a student identified by seat number.
func (sm *StudentManager) UpdateStudent(seatNumber int, name *string, dutyEnabled *bool, lunchEnabled *bool) error {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return err
	}
	for i, s := range students {
		if s.SeatNumber == seatNumber {
			if name != nil {
				students[i].Name = *name
			}
			if dutyEnabled != nil {
				students[i].DutyEnabled = *dutyEnabled
			}
			if lunchEnabled != nil {
				students[i].LunchEnabled = *lunchEnabled
			}
			return sm.dh.SaveStudents(students)
		}
	}
	return fmt.Errorf("找不到座號 %d", seatNumber)
}

// DeleteStudent removes a student by seat number.
func (sm *StudentManager) DeleteStudent(seatNumber int) error {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return err
	}
	filtered := make([]models.Student, 0, len(students))
	for _, s := range students {
		if s.SeatNumber != seatNumber {
			filtered = append(filtered, s)
		}
	}
	return sm.dh.SaveStudents(filtered)
}

// ToggleDuty toggles the duty_enabled flag for a student.
func (sm *StudentManager) ToggleDuty(seatNumber int) error {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return err
	}
	for i, s := range students {
		if s.SeatNumber == seatNumber {
			students[i].DutyEnabled = !s.DutyEnabled
			return sm.dh.SaveStudents(students)
		}
	}
	return fmt.Errorf("找不到座號 %d", seatNumber)
}

// ToggleLunch toggles the lunch_enabled flag for a student.
func (sm *StudentManager) ToggleLunch(seatNumber int) error {
	students, err := sm.dh.GetStudents()
	if err != nil {
		return err
	}
	for i, s := range students {
		if s.SeatNumber == seatNumber {
			students[i].LunchEnabled = !s.LunchEnabled
			return sm.dh.SaveStudents(students)
		}
	}
	return fmt.Errorf("找不到座號 %d", seatNumber)
}
