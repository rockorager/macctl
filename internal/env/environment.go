package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"go.rockorager.dev/macctl/internal/launchd"
	"go.rockorager.dev/macctl/internal/xdg"
)

func UserEnvironmentDir() (string, error) {
	configHome, err := xdg.ConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, "environment.d"), nil
}

func Dirs(scope launchd.Scope) ([]string, error) {
	if scope == launchd.ScopeSystem {
		return []string{"/etc/environment.d"}, nil
	}
	userDir, err := UserEnvironmentDir()
	if err != nil {
		return nil, err
	}
	return []string{userDir}, nil
}

func Load(scope launchd.Scope) (map[string]string, error) {
	dirs, err := Dirs(scope)
	if err != nil {
		return nil, err
	}
	vars := map[string]string{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
				continue
			}
			if err := parseFile(filepath.Join(dir, entry.Name()), vars); err != nil {
				return nil, err
			}
		}
	}
	return vars, nil
}

func parseFile(path string, vars map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !validName(key) {
			return fmt.Errorf("%s:%d: expected valid KEY=VALUE", path, lineNo)
		}
		value, err := expand(raw, vars)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		vars[key] = value
	}
	return scanner.Err()
}

func LoadFile(path string, vars map[string]string) error {
	return parseFile(path, vars)
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 && unicode.IsDigit(r) {
			return false
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func expand(value string, vars map[string]string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '$' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) {
			b.WriteByte('$')
			continue
		}
		if value[i+1] == '{' {
			expanded, next, err := expandBraced(value, i, vars)
			if err != nil {
				return "", err
			}
			b.WriteString(expanded)
			i = next - 1
			continue
		}
		name, next := readName(value, i+1)
		if name == "" {
			b.WriteByte('$')
			continue
		}
		b.WriteString(vars[name])
		i = next - 1
	}
	return b.String(), nil
}

func expandBraced(value string, start int, vars map[string]string) (string, int, error) {
	end := strings.IndexByte(value[start+2:], '}')
	if end < 0 {
		return "", 0, fmt.Errorf("unterminated variable expansion")
	}
	content := value[start+2 : start+2+end]
	next := start + 2 + end + 1
	if name, fallback, ok := strings.Cut(content, ":-"); ok {
		if !validName(name) {
			return "", 0, fmt.Errorf("invalid variable name %q", name)
		}
		if vars[name] == "" {
			return fallback, next, nil
		}
		return vars[name], next, nil
	}
	if name, alternate, ok := strings.Cut(content, ":+"); ok {
		if !validName(name) {
			return "", 0, fmt.Errorf("invalid variable name %q", name)
		}
		if vars[name] != "" {
			return alternate, next, nil
		}
		return "", next, nil
	}
	if !validName(content) {
		return "", 0, fmt.Errorf("invalid variable name %q", content)
	}
	return vars[content], next, nil
}

func readName(value string, start int) (string, int) {
	end := start
	for end < len(value) {
		r := rune(value[end])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		end++
	}
	return value[start:end], end
}

func Apply(scope launchd.Scope, vars map[string]string) error {
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := launchd.Run("setenv", key, vars[key]); err != nil {
			return err
		}
	}
	return nil
}

func LaunchdJob(executable string) launchd.Job {
	return launchd.Job{
		Label:            "dev.macctl.environment",
		ProgramArguments: []string{executable, "--user", "daemon-reload"},
		RunAtLoad:        true,
	}
}
