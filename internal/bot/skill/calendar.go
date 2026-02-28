package skill

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const calendarTimeout = 15 * time.Second

// CalendarCheckInstalled checks if icalBuddy CLI is available.
func CalendarCheckInstalled() bool {
	_, err := exec.LookPath("icalBuddy")
	return err == nil
}

// CalendarToday returns today's events from macOS Calendar.
func CalendarToday(ctx context.Context) (string, error) {
	return calendarEvents(ctx, "today", "today+1", "今日")
}

// CalendarTomorrow returns tomorrow's events.
func CalendarTomorrow(ctx context.Context) (string, error) {
	return calendarEvents(ctx, "today+1", "today+2", "明日")
}

// CalendarWeek returns this week's events.
func CalendarWeek(ctx context.Context) (string, error) {
	return calendarEvents(ctx, "today", "today+7", "本週")
}

func calendarEvents(ctx context.Context, from, to, label string) (string, error) {
	if CalendarCheckInstalled() {
		return icalBuddyEvents(ctx, from, to, label)
	}
	return osascriptEvents(ctx, label)
}

func icalBuddyEvents(ctx context.Context, from, to, label string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, calendarTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "icalBuddy",
		"-f",
		"-b", "• ",
		"-nc",
		"-nrd",
		"eventsFrom:"+from, "to:"+to,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("icalBuddy 執行失敗: %w", err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return fmt.Sprintf("📅 %s行程：無排定事項", label), nil
	}
	return fmt.Sprintf("📅 *%s行程*\n\n%s", label, result), nil
}

func osascriptEvents(ctx context.Context, label string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, calendarTimeout)
	defer cancel()

	script := `
set output to ""
set today to current date
set time of today to 0
set tomorrow to today + 1 * days
tell application "Calendar"
	repeat with cal in calendars
		try
			set evts to (every event of cal whose start date ≥ today and start date < tomorrow)
			repeat with evt in evts
				set evtName to summary of evt
				set evtStart to start date of evt
				set evtEnd to end date of evt
				set output to output & "• " & evtName & " (" & time string of evtStart & " - " & time string of evtEnd & ")" & linefeed
			end repeat
		end try
	end repeat
end tell
return output`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Calendar 讀取失敗（建議安裝 icalBuddy: brew install ical-buddy）: %w", err)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return fmt.Sprintf("📅 %s行程：無排定事項", label), nil
	}
	return fmt.Sprintf("📅 *%s行程*\n\n%s", label, result), nil
}
