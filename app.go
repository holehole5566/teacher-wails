package main

import (
	"context"
	"os"
	"path/filepath"
	"teacher-wails/internal/models"
	"teacher-wails/internal/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct bound to the frontend.
type App struct {
	ctx context.Context
	dh  *services.DataHandler
	sm  *services.StudentManager
	dc  *services.DutyCalculator
	se  *services.ScheduleExporter
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved for runtime calls.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Resolve data directory: prefer ./data next to executable,
	// but in dev mode fall back to working directory.
	exe, err := os.Executable()
	var dataDir string
	if err == nil {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	} else {
		dataDir = "data"
	}

	// If the exe-relative data dir doesn't exist and a local ./data does, use local.
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if wd, wdErr := os.Getwd(); wdErr == nil {
			localData := filepath.Join(wd, "data")
			if _, err := os.Stat(localData); err == nil {
				dataDir = localData
			}
		}
	}

	a.dh, err = services.NewDataHandlerWithPath(dataDir)
	if err != nil {
		panic("無法初始化資料: " + err.Error())
	}
	a.sm = services.NewStudentManager(a.dh)
	a.dc = services.NewDutyCalculator(a.dh, a.sm)
	a.se = services.NewScheduleExporter(a.dc)
}

// GetTodayDuty returns today's duty and lunch assignments.
func (a *App) GetTodayDuty() models.TodayDutyResult {
	result, err := a.dc.GetTodayDuty()
	if err != nil {
		return models.TodayDutyResult{
			DutyStudents:     []models.Student{},
			LunchAssignments: []models.LunchAssignment{},
		}
	}
	return result
}

// GetStudents returns all students sorted by seat number.
func (a *App) GetStudents() []models.Student {
	students, err := a.sm.GetAllStudents()
	if err != nil {
		return []models.Student{}
	}
	return students
}

// AddStudent adds a new student with the given seat number and name.
func (a *App) AddStudent(seatNumber int, name string) error {
	return a.sm.AddStudent(seatNumber, name)
}

// DeleteStudent removes a student by seat number.
func (a *App) DeleteStudent(seatNumber int) error {
	return a.sm.DeleteStudent(seatNumber)
}

// ToggleDuty toggles the duty participation flag for a student.
func (a *App) ToggleDuty(seatNumber int) error {
	return a.sm.ToggleDuty(seatNumber)
}

// ToggleLunch toggles the lunch participation flag for a student.
func (a *App) ToggleLunch(seatNumber int) error {
	return a.sm.ToggleLunch(seatNumber)
}

// GetSettings returns the current settings.
func (a *App) GetSettings() models.Settings {
	settings, err := a.dh.GetSettings()
	if err != nil {
		return models.Settings{}
	}
	return settings
}

// SaveSettings saves the given settings.
func (a *App) SaveSettings(settings models.Settings) error {
	return a.dh.SaveSettings(settings)
}

// GetHolidays returns the holiday list.
func (a *App) GetHolidays() []string {
	holidays, err := a.dh.GetHolidays()
	if err != nil {
		return []string{}
	}
	return holidays
}

// AddHoliday adds a date string to the holiday list.
func (a *App) AddHoliday(dateStr string) error {
	holidays, err := a.dh.GetHolidays()
	if err != nil {
		return err
	}
	for _, h := range holidays {
		if h == dateStr {
			return nil // already exists
		}
	}
	holidays = append(holidays, dateStr)
	return a.dh.SaveHolidays(holidays)
}

// DeleteHoliday removes a date from the holiday list.
func (a *App) DeleteHoliday(dateStr string) error {
	holidays, err := a.dh.GetHolidays()
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(holidays))
	for _, h := range holidays {
		if h != dateStr {
			filtered = append(filtered, h)
		}
	}
	return a.dh.SaveHolidays(filtered)
}

// ClearHolidays removes all holidays.
func (a *App) ClearHolidays() error {
	return a.dh.SaveHolidays([]string{})
}

// ExportSchedule opens a save dialog and writes the CSV schedule to the chosen file.
func (a *App) ExportSchedule() (string, error) {
	csv, err := a.se.ExportCSV(90)
	if err != nil {
		return "", err
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "匯出排程表",
		DefaultFilename: "schedule.csv",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 檔案 (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil // user cancelled
	}

	if err := os.WriteFile(filePath, []byte(csv), 0644); err != nil {
		return "", err
	}
	return filePath, nil
}
