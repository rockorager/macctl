package unit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.rockorager.dev/macctl/internal/launchd"
	"go.rockorager.dev/macctl/internal/xdg"
)

func ConfigDir(scope launchd.Scope) (string, error) {
	if scope == launchd.ScopeSystem {
		return "/etc/xdg/macctl/system", nil
	}
	configHome, err := xdg.ConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, "macctl", "user"), nil
}

func UnitPaths(scope launchd.Scope) ([]string, error) {
	dir, err := ConfigDir(scope)
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
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".service") || strings.HasSuffix(name, ".timer") {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func ResolvePath(scope launchd.Scope, name string) (string, error) {
	if strings.ContainsRune(name, os.PathSeparator) {
		return name, nil
	}
	dir, err := ConfigDir(scope)
	if err != nil {
		return "", err
	}
	candidates := []string{name}
	if !strings.HasSuffix(name, ".service") && !strings.HasSuffix(name, ".timer") {
		candidates = append(candidates, name+".service")
	}
	if strings.Contains(name, "@") {
		for _, candidate := range append([]string{}, candidates...) {
			if suffix := filepath.Ext(candidate); suffix != "" {
				prefix := strings.SplitN(strings.TrimSuffix(candidate, suffix), "@", 2)[0]
				candidates = append(candidates, prefix+"@"+suffix)
			}
		}
	}
	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("unit %q not found in %s", name, dir)
}

func LooksLikePathOrUnitFile(name string) bool {
	return strings.ContainsRune(name, os.PathSeparator) || strings.HasSuffix(name, ".service") || strings.HasSuffix(name, ".timer")
}
