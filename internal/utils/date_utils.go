package utils

import (
	"time"
)

// IsWeekday returns true if the date is Monday–Friday.
func IsWeekday(date time.Time) bool {
	wd := date.Weekday()
	return wd >= time.Monday && wd <= time.Friday
}

// IsHoliday returns true if the date string (YYYY-MM-DD) is in the holiday list.
func IsHoliday(date time.Time, holidays []string) bool {
	dateStr := date.Format("2006-01-02")
	for _, h := range holidays {
		if h == dateStr {
			return true
		}
	}
	return false
}

// IsWorkday returns true if the date is a weekday and not a holiday.
func IsWorkday(date time.Time, holidays []string) bool {
	return IsWeekday(date) && !IsHoliday(date, holidays)
}

// CountWorkdays counts workdays from the day after startDate up to and including endDate.
func CountWorkdays(startDate, endDate time.Time, holidays []string) int {
	count := 0
	current := startDate.AddDate(0, 0, 1)
	for !current.After(endDate) {
		if IsWorkday(current, holidays) {
			count++
		}
		current = current.AddDate(0, 0, 1)
	}
	return count
}

// ParseDate parses a YYYY-MM-DD string into a time.Time (at midnight UTC).
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// FormatDate formats a time.Time as YYYY-MM-DD.
func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}

// Weekdays in Traditional Chinese.
var weekdayNames = []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// FormatDisplayDate returns "YYYY年M月D日 星期X".
func FormatDisplayDate(date time.Time) string {
	wd := weekdayNames[date.Weekday()]
	return date.Format("2006年1月2日") + " " + wd
}
