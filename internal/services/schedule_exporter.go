package services

import (
	"fmt"
	"strings"
	"teacher-wails/internal/utils"
	"time"
)

// ScheduleExporter generates a CSV schedule preview.
type ScheduleExporter struct {
	dc *DutyCalculator
}

// NewScheduleExporter creates a ScheduleExporter.
func NewScheduleExporter(dc *DutyCalculator) *ScheduleExporter {
	return &ScheduleExporter{dc: dc}
}

// ExportCSV generates a 90-day schedule as CSV content (with UTF-8 BOM for Excel).
func (se *ScheduleExporter) ExportCSV(days int) (string, error) {
	if days <= 0 {
		days = 90
	}

	var sb strings.Builder
	// UTF-8 BOM for Excel compatibility
	sb.WriteString("\xEF\xBB\xBF")
	sb.WriteString("日期,星期,值日生,抬餐負責\n")

	weekdayNames := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, i)
		dateStr := utils.FormatDate(date)
		weekday := weekdayNames[date.Weekday()]

		dutyStudents, lunchAssignments, err := se.dc.GetDutyForDate(date)
		if err != nil {
			return "", err
		}

		var dutyStr, lunchStr string
		if dutyStudents == nil {
			dutyStr = "無"
			lunchStr = "無"
		} else {
			dutyParts := make([]string, len(dutyStudents))
			for j, s := range dutyStudents {
				dutyParts[j] = fmt.Sprintf("%d號%s", s.SeatNumber, s.Name)
			}
			dutyStr = strings.Join(dutyParts, "、")

			lunchParts := make([]string, len(lunchAssignments))
			for j, a := range lunchAssignments {
				lunchParts[j] = fmt.Sprintf("%d號%s(%s)", a.Student.SeatNumber, a.Student.Name, a.Bucket)
			}
			lunchStr = strings.Join(lunchParts, "、")
		}

		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", dateStr, weekday, dutyStr, lunchStr))
	}

	return sb.String(), nil
}
