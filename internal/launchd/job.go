package launchd

import (
	"os"

	"howett.net/plist"
)

type CalendarInterval struct {
	Second  *int `plist:"Second,omitempty"`
	Minute  *int `plist:"Minute,omitempty"`
	Hour    *int `plist:"Hour,omitempty"`
	Day     *int `plist:"Day,omitempty"`
	Weekday *int `plist:"Weekday,omitempty"`
	Month   *int `plist:"Month,omitempty"`
}

type Job struct {
	Label                 string            `plist:"Label"`
	ProgramArguments      []string          `plist:"ProgramArguments,omitempty"`
	WorkingDirectory      string            `plist:"WorkingDirectory,omitempty"`
	EnvironmentVariables  map[string]string `plist:"EnvironmentVariables,omitempty"`
	RunAtLoad             bool              `plist:"RunAtLoad,omitempty"`
	KeepAlive             any               `plist:"KeepAlive,omitempty"`
	ThrottleInterval      *int              `plist:"ThrottleInterval,omitempty"`
	StartInterval         *int              `plist:"StartInterval,omitempty"`
	StartCalendarInterval any               `plist:"StartCalendarInterval,omitempty"`
	StandardOutPath       string            `plist:"StandardOutPath,omitempty"`
	StandardErrorPath     string            `plist:"StandardErrorPath,omitempty"`
}

func WritePlist(path string, job Job) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := plist.NewEncoder(f)
	enc.Indent("\t")
	return enc.Encode(job)
}
