package systemdsyntax

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

type Context struct {
	UnitName     string
	FragmentPath string
	Environment  map[string]string
}

func ParseCommandLine(value string, ctx Context) ([]string, error) {
	items, err := SplitItems(value)
	if err != nil {
		return nil, err
	}
	var argv []string
	for _, item := range items {
		expanded, err := ExpandSpecifiers(item, ctx)
		if err != nil {
			return nil, err
		}
		expandedItems, err := expandCommandEnvironment(expanded, ctx.Environment)
		if err != nil {
			return nil, err
		}
		argv = append(argv, expandedItems...)
	}
	return argv, nil
}

func ExpandSpecifiers(value string, ctx Context) (string, error) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) {
			return "", fmt.Errorf("trailing percent specifier")
		}
		replacement, err := specifierValue(value[i+1], ctx)
		if err != nil {
			return "", err
		}
		b.WriteString(replacement)
		i++
	}
	return b.String(), nil
}

func specifierValue(spec byte, ctx Context) (string, error) {
	u, _ := user.Current()
	hostname, _ := os.Hostname()
	unit := splitUnitName(ctx.UnitName)
	switch spec {
	case '%':
		return "%", nil
	case 'a':
		return architecture(), nil
	case 'A', 'b', 'B', 'm', 'M', 'o', 'v', 'w', 'W':
		return "", nil
	case 'C':
		return xdgOrDefault("XDG_CACHE_HOME", ".cache", u), nil
	case 'd':
		return os.Getenv("CREDENTIALS_DIRECTORY"), nil
	case 'D':
		return xdgOrDefault("XDG_DATA_HOME", filepath.Join(".local", "share"), u), nil
	case 'E':
		return xdgOrDefault("XDG_CONFIG_HOME", ".config", u), nil
	case 'f':
		if unit.Instance != "" {
			return "/" + unit.Instance, nil
		}
		return "/" + unit.Prefix, nil
	case 'g', 'u':
		if u != nil {
			return u.Username, nil
		}
		return "root", nil
	case 'G':
		if u != nil {
			return u.Gid, nil
		}
		return "0", nil
	case 'h':
		if u != nil {
			return u.HomeDir, nil
		}
		return "/var/root", nil
	case 'H':
		return hostname, nil
	case 'i', 'I', 'j', 'J':
		return unit.Instance, nil
	case 'l':
		if idx := strings.IndexByte(hostname, '.'); idx >= 0 {
			return hostname[:idx], nil
		}
		return hostname, nil
	case 'L', 'S':
		return xdgOrDefault("XDG_STATE_HOME", filepath.Join(".local", "state"), u), nil
	case 'n':
		return ctx.UnitName, nil
	case 'N':
		return unit.NameWithoutSuffix, nil
	case 'p', 'P':
		return unit.Prefix, nil
	case 's':
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell, nil
		}
		return "/bin/sh", nil
	case 't':
		return runtimeDir(u), nil
	case 'T':
		return os.TempDir(), nil
	case 'U':
		if u != nil {
			return u.Uid, nil
		}
		return "0", nil
	case 'V':
		return "/var/tmp", nil
	case 'y':
		if ctx.FragmentPath == "" {
			return "", fmt.Errorf("%%y requires a fragment path")
		}
		return ctx.FragmentPath, nil
	case 'Y':
		if ctx.FragmentPath == "" {
			return "", fmt.Errorf("%%Y requires a fragment path")
		}
		return filepath.Dir(ctx.FragmentPath), nil
	default:
		return "", fmt.Errorf("unsupported percent specifier %%%c", spec)
	}
}

func expandCommandEnvironment(value string, env map[string]string) ([]string, error) {
	if len(value) > 1 && value[0] == '$' && value[1] != '{' {
		name := value[1:]
		if validEnvironmentName(name) {
			if env[name] == "" {
				return nil, nil
			}
			return SplitItems(env[name])
		}
	}
	expanded, err := expandBracedEnvironment(value, env)
	if err != nil {
		return nil, err
	}
	return []string{expanded}, nil
}

func expandBracedEnvironment(value string, env map[string]string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '$' || i+1 >= len(value) || value[i+1] != '{' {
			b.WriteByte(value[i])
			continue
		}
		end := strings.IndexByte(value[i+2:], '}')
		if end < 0 {
			return "", fmt.Errorf("unterminated environment expansion")
		}
		name := value[i+2 : i+2+end]
		if !validEnvironmentName(name) {
			return "", fmt.Errorf("invalid environment variable name %q", name)
		}
		b.WriteString(env[name])
		i = i + 2 + end
	}
	return b.String(), nil
}

func validEnvironmentName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if i == 0 && c >= '0' && c <= '9' {
			return false
		}
		if !isEnvironmentNameChar(c) {
			return false
		}
	}
	return true
}

func isEnvironmentNameChar(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

type unitName struct {
	NameWithoutSuffix string
	Prefix            string
	Instance          string
}

func splitUnitName(name string) unitName {
	base := name
	if idx := strings.LastIndexByte(base, '.'); idx >= 0 {
		base = base[:idx]
	}
	prefix := base
	instance := ""
	if before, after, ok := strings.Cut(base, "@"); ok {
		prefix = before
		instance = after
	}
	return unitName{NameWithoutSuffix: base, Prefix: prefix, Instance: instance}
}

func architecture() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86-64"
	case "386":
		return "x86"
	default:
		return runtime.GOARCH
	}
}

func xdgOrDefault(name, relativeDefault string, u *user.User) string {
	if value := os.Getenv(name); value != "" && filepath.IsAbs(value) {
		return value
	}
	if u == nil {
		return filepath.Join("/", relativeDefault)
	}
	return filepath.Join(u.HomeDir, relativeDefault)
}

func runtimeDir(u *user.User) string {
	if value := os.Getenv("XDG_RUNTIME_DIR"); value != "" && filepath.IsAbs(value) {
		return value
	}
	if u != nil {
		return filepath.Join(os.TempDir(), "macctl-runtime-"+u.Uid)
	}
	return os.TempDir()
}
