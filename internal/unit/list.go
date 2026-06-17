package unit

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.rockorager.dev/macctl/internal/launchd"
)

type FileState struct {
	Name  string
	Path  string
	State string
}

func UnitFiles(scope launchd.Scope) ([]FileState, error) {
	paths, err := UnitPaths(scope)
	if err != nil {
		return nil, err
	}
	files := make([]FileState, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		state := "disabled"
		if enabled, err := GeneratedPlistRunAtLoad(scope, name); err != nil {
			return nil, err
		} else if enabled {
			state = "enabled"
		}
		files = append(files, FileState{Name: name, Path: path, State: state})
	}
	return files, nil
}

func ConfigUnitNames(scope launchd.Scope) ([]string, error) {
	paths, err := UnitPaths(scope)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}
	return names, nil
}

func GeneratedUnitNames(scope launchd.Scope) ([]string, error) {
	dir, err := launchd.PlistDir(scope)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "dev.macctl.") || !strings.HasSuffix(name, ".plist") {
			continue
		}
		name = strings.TrimPrefix(strings.TrimSuffix(name, ".plist"), "dev.macctl.")
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func EnabledUnitNames(scope launchd.Scope) ([]string, error) {
	files, err := UnitFiles(scope)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, file := range files {
		if file.State == "enabled" {
			names = append(names, file.Name)
		}
	}
	return names, nil
}

func GeneratedPlistRunAtLoad(scope launchd.Scope, unitName string) (bool, error) {
	dir, err := launchd.PlistDir(scope)
	if err != nil {
		return false, err
	}
	path := filepath.Join(dir, "dev.macctl."+LabelName(unitName)+".plist")
	job, err := launchd.ReadPlist(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return job.RunAtLoad, nil
}

func LabelName(name string) string {
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".service"), ".timer")
	return name
}
