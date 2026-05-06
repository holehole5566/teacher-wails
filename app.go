package main

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"math/rand"
	"teacher-wails/internal/models"
	"teacher-wails/internal/services"
	"time"

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

// SyncHolidays fetches the current year's government holiday calendar from
// data.gov.tw and merges non-weekend holidays into the local holiday list.
func (a *App) SyncHolidays() (int, error) {
	year := time.Now().Year()
	rocYear := year - 1911

	// 1. Query data.gov.tw API to find the CSV download URL for this ROC year.
	csvURL, err := findHolidayCSVURL(rocYear)
	if err != nil {
		return 0, fmt.Errorf("找不到 %d 年辦公日曆表: %w", rocYear, err)
	}

	// 2. Download and parse the CSV.
	dates, err := fetchNonWeekendHolidays(csvURL)
	if err != nil {
		return 0, fmt.Errorf("下載或解析 CSV 失敗: %w", err)
	}

	// 3. Merge into existing holidays.
	holidays, _ := a.dh.GetHolidays()
	existing := make(map[string]bool, len(holidays))
	for _, h := range holidays {
		existing[h] = true
	}
	added := 0
	for _, d := range dates {
		if !existing[d] {
			holidays = append(holidays, d)
			added++
		}
	}
	sort.Strings(holidays)
	if err := a.dh.SaveHolidays(holidays); err != nil {
		return 0, err
	}
	return added, nil
}

// findHolidayCSVURL queries the data.gov.tw API for dataset 14718 and returns
// the download URL for the CSV matching the given ROC year.
func findHolidayCSVURL(rocYear int) (string, error) {
	resp, err := http.Get("https://data.gov.tw/api/v2/rest/dataset/14718")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Distribution []struct {
				ResourceDescription string `json:"resourceDescription"`
				ResourceDownloadUrl string `json:"resourceDownloadUrl"`
			} `json:"distribution"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	yearStr := fmt.Sprintf("%d", rocYear)
	// Find the best match: description contains the year, is not Google calendar, not 更新 version
	var fallback string
	for _, d := range result.Result.Distribution {
		desc := d.ResourceDescription
		if !strings.Contains(desc, yearStr) {
			continue
		}
		if strings.Contains(desc, "Google") {
			continue
		}
		// Prefer the updated version if available
		if strings.Contains(desc, "更新") {
			return d.ResourceDownloadUrl, nil
		}
		fallback = d.ResourceDownloadUrl
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no CSV found for ROC year %d", rocYear)
}

// fetchNonWeekendHolidays downloads a government holiday CSV and returns
// dates (YYYY-MM-DD) that are holidays falling on weekdays (Mon-Fri).
func fetchNonWeekendHolidays(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Strip BOM if present
	content := strings.TrimPrefix(string(body), "\xef\xbb\xbf")
	reader := csv.NewReader(strings.NewReader(content))

	var dates []string
	header := true
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if header {
			header = false
			continue
		}
		if len(record) < 4 {
			continue
		}
		dateRaw := strings.TrimSpace(record[0]) // e.g. "20260101"
		dayOfWeek := strings.TrimSpace(record[1])
		isHoliday := strings.TrimSpace(record[2])

		if isHoliday != "2" {
			continue
		}
		// Skip weekends (六=Saturday, 日=Sunday)
		if dayOfWeek == "六" || dayOfWeek == "日" {
			continue
		}
		if len(dateRaw) != 8 {
			continue
		}
		// Convert YYYYMMDD → YYYY-MM-DD
		formatted := dateRaw[:4] + "-" + dateRaw[4:6] + "-" + dateRaw[6:8]
		dates = append(dates, formatted)
	}
	return dates, nil
}

// ExportSchedule opens a save dialog and writes the CSV schedule to the chosen file.
func (a *App) ExportSchedule() (string, error) {
	csvData, err := a.se.ExportCSV(90)
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

	if err := os.WriteFile(filePath, []byte(csvData), 0644); err != nil {
		return "", err
	}
	return filePath, nil
}


// SetFullscreen toggles the window fullscreen state.
func (a *App) SetFullscreen(enabled bool) {
	if enabled {
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowShow(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowFullscreen(a.ctx)
	} else {
		runtime.WindowSetAlwaysOnTop(a.ctx, false)
		runtime.WindowUnfullscreen(a.ctx)
	}
}

// GetTimetable returns the 5×8 timetable.
func (a *App) GetTimetable() [5][8]string {
	config, err := a.dh.LoadConfig()
	if err != nil {
		return [5][8]string{}
	}
	return config.Timetable
}

// SaveTimetable saves the 5×8 timetable.
func (a *App) SaveTimetable(timetable [5][8]string) error {
	return a.dh.SaveTimetable(timetable)
}

// SelectCountdownMusics opens a multi-file dialog for the user to pick MP3 files.
func (a *App) SelectCountdownMusics() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "選擇倒數音樂",
		Filters: []runtime.FileFilter{
			{DisplayName: "MP3 音訊 (*.mp3)", Pattern: "*.mp3"},
		},
	})
}

// GetCountdownMusicData reads CountdownMusics[index] and returns a base64 data URL.
// Used by the settings page for per-track preview.
func (a *App) GetCountdownMusicData(index int) (string, error) {
	settings, err := a.dh.GetSettings()
	if err != nil {
		return "", err
	}
	if index < 0 || index >= len(settings.CountdownMusics) {
		return "", fmt.Errorf("音樂索引超出範圍")
	}
	return a.readMusicFile(settings.CountdownMusics[index].Path)
}

// GetActiveCountdownMusicData selects and returns a base64 data URL for the track
// to play at the given countdown time. Random mode picks from InRandom tracks;
// index mode picks the specified track.
func (a *App) GetActiveCountdownMusicData(triggerTime string) (string, error) {
	settings, err := a.dh.GetSettings()
	if err != nil {
		return "", err
	}
	if len(settings.CountdownMusics) == 0 {
		return "", nil
	}

	mode := "random"
	selectedIndex := -1
	for _, ctm := range settings.CountdownTimeMusicMap {
		if ctm.Time == triggerTime {
			mode = ctm.Mode
			selectedIndex = ctm.Index
			break
		}
	}

	if mode == "none" {
		return "", nil
	}

	if mode == "index" {
		if selectedIndex < 0 || selectedIndex >= len(settings.CountdownMusics) {
			return "", fmt.Errorf("指定的音樂索引無效")
		}
		return a.readMusicFile(settings.CountdownMusics[selectedIndex].Path)
	}

	var pool []string
	for _, t := range settings.CountdownMusics {
		if t.InRandom {
			pool = append(pool, t.Path)
		}
	}
	if len(pool) == 0 {
		return "", fmt.Errorf("隨機清單為空")
	}
	return a.readMusicFile(pool[rand.Intn(len(pool))])
}

// ValidateRandomPool returns true if at least one track has InRandom==true and its file exists.
func (a *App) ValidateRandomPool() bool {
	settings, err := a.dh.GetSettings()
	if err != nil {
		return false
	}
	for _, t := range settings.CountdownMusics {
		if t.InRandom {
			if _, err := os.Stat(t.Path); err == nil {
				return true
			}
		}
	}
	return false
}

// readMusicFile reads a file path, validates it as MP3, and returns a base64 data URL.
func (a *App) readMusicFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !isMP3(data) {
		return "", fmt.Errorf("檔案不是有效的 MP3 格式")
	}
	return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
}

// ReportError sends an error message to the configured Discord webhook.
func (a *App) ReportError(msg string) {
	settings, err := a.dh.GetSettings()
	if err != nil || settings.DiscordWebhook == "" {
		return
	}
	hostname, _ := os.Hostname()
	body, _ := json.Marshal(map[string]string{
		"content": fmt.Sprintf("🚨 **TeacherApp 錯誤**\n機器：`%s`\n時間：`%s`\n錯誤：%s", hostname, time.Now().Format("2006-01-02 15:04:05"), msg),
	})
	resp, err := http.Post(settings.DiscordWebhook, "application/json", strings.NewReader(string(body)))
	if err == nil {
		resp.Body.Close()
	}
}

// isMP3 checks if data starts with valid MP3 magic bytes (ID3 tag or MPEG sync word).
func isMP3(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	// ID3v2 tag
	if data[0] == 'I' && data[1] == 'D' && data[2] == '3' {
		return true
	}
	// MPEG audio frame sync: 0xFF followed by 0xE0+ (11 sync bits set)
	if len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0 {
		return true
	}
	return false
}
