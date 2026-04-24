package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

var absoluteLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"2006/01/02 15:04",
	"2006/01/02",
	"02.01.2006 15:04",
	"02.01.2006",
	"02/01/2006 15:04",
	"02/01/2006",
}

var (
	timeTokenRe  = regexp.MustCompile(`^\d{1,2}(:\d{2})?(am|pm|a|p)?$`)
	relPlusRe    = regexp.MustCompile(`^\+(\d+)\s*(h|hr|hrs|hour|hours|m|min|mins|minute|minutes|d|day|days|w|wk|week|weeks)$`)
	relWordRe    = regexp.MustCompile(`^in\s+(\d+)\s*(h|hr|hrs|hour|hours|m|min|mins|minute|minutes|d|day|days|w|wk|week|weeks)$`)
	relCompactRe = regexp.MustCompile(`^(\d+)(h|m|d|w)$`)
)

// ParseDue parses a natural-language due date relative to now.
func ParseDue(input string) (time.Time, error) {
	return parseDueAt(input, time.Now())
}

func parseDueAt(input string, now time.Time) (time.Time, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return time.Time{}, fmt.Errorf("empty due date")
	}

	for _, layout := range absoluteLayouts {
		if t, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return t, nil
		}
	}

	if d, ok := parseRelative(s); ok {
		return now.Add(d), nil
	}

	forceNextWeek := false
	if strings.HasPrefix(s, "next ") {
		forceNextWeek = true
		s = strings.TrimPrefix(s, "next ")
	} else if strings.HasPrefix(s, "this ") {
		s = strings.TrimPrefix(s, "this ")
	}

	dayPart, timePart := splitDayAndTime(strings.Fields(s))

	var base time.Time
	switch dayPart {
	case "", "today":
		base = startOfDay(now)
	case "tomorrow", "tmrw", "tom":
		base = startOfDay(now).AddDate(0, 0, 1)
	case "tonight":
		base = startOfDay(now)
		if timePart == "" {
			timePart = "20:00"
		}
	case "yesterday":
		base = startOfDay(now).AddDate(0, 0, -1)
	default:
		if wd, ok := weekdays[dayPart]; ok {
			base = nextWeekday(startOfDay(now), wd, forceNextWeek)
		} else {
			return time.Time{}, fmt.Errorf("unrecognized due date: %q", input)
		}
	}

	if timePart != "" {
		h, m, err := parseClock(timePart)
		if err != nil {
			return time.Time{}, fmt.Errorf("unrecognized time %q in %q", timePart, input)
		}
		base = time.Date(base.Year(), base.Month(), base.Day(), h, m, 0, 0, base.Location())
	}

	// Bare time like "2pm" without a day: if already past, bump to tomorrow.
	if dayPart == "" && timePart != "" && base.Before(now) {
		base = base.AddDate(0, 0, 1)
	}

	return base, nil
}

func splitDayAndTime(parts []string) (day, timeStr string) {
	for i, p := range parts {
		if timeTokenRe.MatchString(p) {
			day = strings.Join(parts[:i], " ")
			timeStr = strings.Join(parts[i:], " ")
			return
		}
		// Handle "2 pm" style: digits followed by "am"/"pm".
		if isAllDigits(p) && i+1 < len(parts) && (parts[i+1] == "am" || parts[i+1] == "pm") {
			day = strings.Join(parts[:i], " ")
			timeStr = strings.Join(parts[i:], "")
			return
		}
	}
	day = strings.Join(parts, " ")
	return
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseClock(s string) (int, int, error) {
	s = strings.ReplaceAll(s, " ", "")
	ampm := ""
	switch {
	case strings.HasSuffix(s, "pm"):
		ampm = "pm"
		s = strings.TrimSuffix(s, "pm")
	case strings.HasSuffix(s, "am"):
		ampm = "am"
		s = strings.TrimSuffix(s, "am")
	case strings.HasSuffix(s, "p"):
		ampm = "pm"
		s = strings.TrimSuffix(s, "p")
	case strings.HasSuffix(s, "a"):
		ampm = "am"
		s = strings.TrimSuffix(s, "a")
	}

	var hour, min int
	var err error
	if idx := strings.Index(s, ":"); idx >= 0 {
		hour, err = strconv.Atoi(s[:idx])
		if err != nil {
			return 0, 0, err
		}
		min, err = strconv.Atoi(s[idx+1:])
		if err != nil {
			return 0, 0, err
		}
	} else {
		hour, err = strconv.Atoi(s)
		if err != nil {
			return 0, 0, err
		}
	}

	if ampm == "pm" && hour < 12 {
		hour += 12
	} else if ampm == "am" && hour == 12 {
		hour = 0
	}

	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("invalid time")
	}
	return hour, min, nil
}

func parseRelative(s string) (time.Duration, bool) {
	if m := relPlusRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return toDuration(n, m[2]), true
	}
	if m := relWordRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return toDuration(n, m[2]), true
	}
	if m := relCompactRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return toDuration(n, m[2]), true
	}
	return 0, false
}

func toDuration(n int, unit string) time.Duration {
	switch unit[0] {
	case 'h':
		return time.Duration(n) * time.Hour
	case 'm':
		return time.Duration(n) * time.Minute
	case 'd':
		return time.Duration(n) * 24 * time.Hour
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour
	}
	return 0
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func nextWeekday(from time.Time, target time.Weekday, forceNextWeek bool) time.Time {
	diff := (int(target) - int(from.Weekday()) + 7) % 7
	if diff == 0 && forceNextWeek {
		diff = 7
	}
	return from.AddDate(0, 0, diff)
}

// formatDue returns a short human label and an ANSI color for the due date.
func formatDue(due, now time.Time) (string, string) {
	delta := due.Sub(now)
	midnight := startOfDay(now)
	dueDay := startOfDay(due)
	daysDiff := int(dueDay.Sub(midnight).Hours() / 24)
	hasTime := due.Hour() != 0 || due.Minute() != 0

	if delta < 0 {
		return "overdue " + humanDuration(-delta) + " ago", red
	}

	var label string
	switch daysDiff {
	case 0:
		if hasTime {
			label = "today " + due.Format("15:04")
		} else {
			label = "today"
		}
	case 1:
		if hasTime {
			label = "tomorrow " + due.Format("15:04")
		} else {
			label = "tomorrow"
		}
	default:
		if daysDiff > 1 && daysDiff < 7 {
			if hasTime {
				label = due.Format("Mon 15:04")
			} else {
				label = due.Format("Mon")
			}
		} else if hasTime {
			label = due.Format("2006-01-02 15:04")
		} else {
			label = due.Format("2006-01-02")
		}
	}

	color := dim
	if delta < 24*time.Hour {
		color = yellow
	}
	if daysDiff == 0 && hasTime && delta < 2*time.Hour {
		color = boldRed
	}
	return "due " + label, color
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
