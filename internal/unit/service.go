package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
	envd "go.rockorager.dev/macctl/internal/env"
	"go.rockorager.dev/macctl/internal/launchd"
	"go.rockorager.dev/macctl/internal/systemdsyntax"
)

type Service struct {
	Name             string
	Description      string
	ExecStart        []string
	WorkingDirectory string
	Environment      map[string]string
	Restart          string
	RestartSec       *int
}

func LoadService(path string) (*Service, error) {
	return LoadServiceAs(path, filepath.Base(path))
}

func LoadServiceAs(path, unitName string) (*Service, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	opts, err := unit.DeserializeOptions(f)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		Name:        strings.TrimSuffix(unitName, filepath.Ext(unitName)),
		Environment: map[string]string{},
	}
	ctx := systemdsyntax.Context{
		UnitName:     unitName,
		FragmentPath: path,
		Environment:  svc.Environment,
	}
	execStart := ""
	for _, opt := range opts {
		switch opt.Section + "." + opt.Name {
		case "Unit.Description":
			svc.Description = opt.Value
		case "Service.ExecStart":
			execStart = opt.Value
		case "Service.WorkingDirectory":
			svc.WorkingDirectory = opt.Value
		case "Service.Environment":
			for _, assignment := range splitEnvironment(opt.Value) {
				key, value, ok := strings.Cut(assignment, "=")
				if !ok || key == "" {
					return nil, fmt.Errorf("invalid Environment assignment %q", assignment)
				}
				svc.Environment[key] = value
			}
		case "Service.EnvironmentFile":
			for _, path := range splitEnvironment(opt.Value) {
				optional := strings.HasPrefix(path, "-")
				path = strings.TrimPrefix(path, "-")
				if err := envd.LoadFile(path, svc.Environment); err != nil && (!optional || !os.IsNotExist(err)) {
					return nil, fmt.Errorf("load EnvironmentFile %q: %w", path, err)
				}
			}
		case "Service.Restart":
			svc.Restart = opt.Value
		case "Service.RestartSec":
			seconds, err := parseSeconds(opt.Value)
			if err != nil {
				return nil, fmt.Errorf("parse RestartSec: %w", err)
			}
			svc.RestartSec = &seconds
		}
	}
	if execStart != "" {
		argv, err := systemdsyntax.ParseCommandLine(execStart, ctx)
		if err != nil {
			return nil, fmt.Errorf("parse ExecStart: %w", err)
		}
		svc.ExecStart = argv
	}
	if len(svc.ExecStart) == 0 {
		return nil, fmt.Errorf("%s: Service.ExecStart is required", path)
	}
	return svc, nil
}

func (s *Service) LaunchdJob() launchd.Job {
	job := launchd.Job{
		Label:                "dev.macctl." + s.Name,
		ProgramArguments:     s.ExecStart,
		WorkingDirectory:     s.WorkingDirectory,
		EnvironmentVariables: s.Environment,
	}
	if s.Restart == "always" || s.Restart == "on-failure" {
		job.KeepAlive = true
	}
	if s.RestartSec != nil {
		job.ThrottleInterval = s.RestartSec
	}
	return job
}

func splitEnvironment(value string) []string {
	fields, err := systemdsyntax.SplitItems(value)
	if err != nil || len(fields) == 0 {
		return []string{value}
	}
	return fields
}
