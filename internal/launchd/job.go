package launchd

import (
	"bytes"
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
	b, err := MarshalPlist(job)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func WritePlistIfChanged(path string, job Job) (bool, error) {
	b, err := MarshalPlist(job)
	if err != nil {
		return false, err
	}
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, b) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, os.WriteFile(path, b, 0o644)
}

func MarshalPlist(job Job) ([]byte, error) {
	var b bytes.Buffer
	enc := plist.NewEncoder(&b)
	enc.Indent("\t")
	if err := enc.Encode(job); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func ReadPlist(path string) (Job, error) {
	f, err := os.Open(path)
	if err != nil {
		return Job{}, err
	}
	defer func() { _ = f.Close() }()
	var job Job
	if err := plist.NewDecoder(f).Decode(&job); err != nil {
		return Job{}, err
	}
	return job, nil
}
