package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
	"go.rockorager.dev/macctl/internal/launchd"
)

type Timer struct {
	Name             string
	Description      string
	Unit             string
	OnCalendar       []launchd.CalendarInterval
	StartIntervalSec *int
	RunAtLoad        bool
	Service          *Service
}

func LoadTimer(path string) (*Timer, error) {
	return LoadTimerAs(path, filepath.Base(path))
}

func LoadTimerAs(path, unitName string) (*Timer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	opts, err := unit.DeserializeOptions(f)
	if err != nil {
		return nil, err
	}
	t := &Timer{
		Name: strings.TrimSuffix(unitName, filepath.Ext(unitName)),
		Unit: strings.TrimSuffix(unitName, ".timer") + ".service",
	}
	for _, opt := range opts {
		switch opt.Section + "." + opt.Name {
		case "Unit.Description":
			t.Description = opt.Value
		case "Timer.Unit":
			t.Unit = opt.Value
		case "Timer.OnCalendar":
			intervals, err := parseOnCalendar(opt.Value)
			if err != nil {
				return nil, err
			}
			t.OnCalendar = append(t.OnCalendar, intervals...)
		case "Timer.OnActiveSec", "Timer.OnUnitActiveSec", "Timer.OnUnitInactiveSec":
			seconds, err := parseSeconds(opt.Value)
			if err != nil {
				return nil, err
			}
			t.StartIntervalSec = &seconds
		case "Timer.OnBootSec", "Timer.OnStartupSec":
			if _, err := parseSeconds(opt.Value); err != nil {
				return nil, err
			}
			t.RunAtLoad = true
		case "Timer.AccuracySec", "Timer.RandomizedDelaySec", "Timer.RandomizedOffsetSec":
			if _, err := parseSeconds(opt.Value); err != nil {
				return nil, err
			}
		case "Timer.FixedRandomDelay", "Timer.DeferReactivation", "Timer.OnClockChange", "Timer.OnTimezoneChange", "Timer.Persistent", "Timer.WakeSystem", "Timer.RemainAfterElapse":
			// Valid systemd.timer keys. launchd has no direct equivalent in the
			// generated plist today, so macctl accepts them without changing output.
		}
	}
	servicePath := filepath.Join(filepath.Dir(path), t.Unit)
	svc, err := LoadServiceAs(servicePath, t.Unit)
	if err != nil {
		return nil, err
	}
	t.Service = svc
	return t, nil
}

func (t *Timer) LaunchdJob() launchd.Job {
	job := t.Service.LaunchdJob()
	job.Label = "dev.macctl." + t.Name
	job.RunAtLoad = t.RunAtLoad
	job.KeepAlive = nil
	job.ThrottleInterval = nil
	job.StartInterval = t.StartIntervalSec
	if len(t.OnCalendar) == 1 {
		job.StartCalendarInterval = t.OnCalendar[0]
	} else if len(t.OnCalendar) > 1 {
		job.StartCalendarInterval = t.OnCalendar
	}
	return job
}

func parseOnCalendar(value string) ([]launchd.CalendarInterval, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "minutely":
		return []launchd.CalendarInterval{{Second: intPtr(0)}}, nil
	case "hourly":
		return []launchd.CalendarInterval{{Minute: intPtr(0)}}, nil
	case "daily", "midnight":
		return []launchd.CalendarInterval{{Hour: intPtr(0), Minute: intPtr(0)}}, nil
	case "weekly":
		return []launchd.CalendarInterval{{Weekday: intPtr(0), Hour: intPtr(0), Minute: intPtr(0)}}, nil
	case "monthly":
		return []launchd.CalendarInterval{{Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)}}, nil
	case "quarterly":
		return []launchd.CalendarInterval{
			{Month: intPtr(1), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
			{Month: intPtr(4), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
			{Month: intPtr(7), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
			{Month: intPtr(10), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
		}, nil
	case "semiannually", "semi-annually":
		return []launchd.CalendarInterval{
			{Month: intPtr(1), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
			{Month: intPtr(7), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)},
		}, nil
	case "annually", "yearly", "anually":
		return []launchd.CalendarInterval{{Month: intPtr(1), Day: intPtr(1), Hour: intPtr(0), Minute: intPtr(0)}}, nil
	}
	datePart, timePart := splitCalendar(value)
	base, err := parseCalendarTime(timePart)
	if err != nil {
		return nil, err
	}
	if datePart == "" || datePart == "*-*-*" {
		return []launchd.CalendarInterval{base}, nil
	}
	if intervals, ok, err := parseWeekdayCalendar(datePart, base); ok || err != nil {
		return intervals, err
	}
	if interval, ok, err := parseDateCalendar(datePart, base); ok || err != nil {
		return []launchd.CalendarInterval{interval}, err
	}
	return nil, fmt.Errorf("unsupported OnCalendar %q", value)
}

func splitCalendar(value string) (string, string) {
	fields := strings.Fields(value)
	if len(fields) == 1 {
		return "", fields[0]
	}
	return strings.Join(fields[:len(fields)-1], " "), fields[len(fields)-1]
}

func parseCalendarTime(value string) (launchd.CalendarInterval, error) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return launchd.CalendarInterval{}, fmt.Errorf("unsupported calendar time %q", value)
	}
	h, err := atoiWildcard(parts[0])
	if err != nil {
		return launchd.CalendarInterval{}, err
	}
	m, err := atoiWildcard(parts[1])
	if err != nil {
		return launchd.CalendarInterval{}, err
	}
	interval := launchd.CalendarInterval{Hour: h, Minute: m}
	if len(parts) == 3 {
		s, err := atoiWildcard(parts[2])
		if err != nil {
			return launchd.CalendarInterval{}, err
		}
		interval.Second = s
	}
	return interval, nil
}

func parseWeekdayCalendar(value string, base launchd.CalendarInterval) ([]launchd.CalendarInterval, bool, error) {
	weekdays, err := parseWeekdays(value)
	if err != nil {
		return nil, true, err
	}
	if len(weekdays) == 0 {
		return nil, false, nil
	}
	intervals := make([]launchd.CalendarInterval, 0, len(weekdays))
	for _, weekday := range weekdays {
		interval := base
		interval.Weekday = &weekday
		intervals = append(intervals, interval)
	}
	return intervals, true, nil
}

func parseDateCalendar(value string, base launchd.CalendarInterval) (launchd.CalendarInterval, bool, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 3 {
		return launchd.CalendarInterval{}, false, nil
	}
	month, err := atoiWildcard(parts[1])
	if err != nil {
		return launchd.CalendarInterval{}, true, err
	}
	day, err := atoiWildcard(parts[2])
	if err != nil {
		return launchd.CalendarInterval{}, true, err
	}
	base.Month = month
	base.Day = day
	return base, true, nil
}

func parseWeekdays(value string) ([]int, error) {
	var weekdays []int
	for _, field := range strings.Split(value, ",") {
		start, end, rangeOK := strings.Cut(field, "..")
		if !rangeOK {
			start, end, rangeOK = strings.Cut(field, "-")
		}
		startDay, ok := weekdayNumber(start)
		if !ok {
			return nil, nil
		}
		if !rangeOK {
			weekdays = append(weekdays, startDay)
			continue
		}
		endDay, ok := weekdayNumber(end)
		if !ok {
			return nil, fmt.Errorf("invalid weekday %q", end)
		}
		for day := startDay; ; day = (day + 1) % 7 {
			weekdays = append(weekdays, day)
			if day == endDay {
				break
			}
		}
	}
	return weekdays, nil
}

func weekdayNumber(value string) (int, bool) {
	switch strings.ToLower(value) {
	case "sun", "sunday":
		return 0, true
	case "mon", "monday":
		return 1, true
	case "tue", "tuesday":
		return 2, true
	case "wed", "wednesday":
		return 3, true
	case "thu", "thursday":
		return 4, true
	case "fri", "friday":
		return 5, true
	case "sat", "saturday":
		return 6, true
	default:
		return 0, false
	}
}

func atoiWildcard(value string) (*int, error) {
	if value == "*" {
		return nil, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func intPtr(v int) *int { return &v }
