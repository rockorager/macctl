package launchd

import (
	"bytes"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

func Domain(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		return "gui/" + u.Uid, nil
	case ScopeSystem:
		return "system", nil
	default:
		return "", fmt.Errorf("unknown scope %q", scope)
	}
}

func PlistDir(scope Scope) (string, error) {
	switch scope {
	case ScopeUser:
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(u.HomeDir, "Library", "LaunchAgents"), nil
	case ScopeSystem:
		return "/Library/LaunchDaemons", nil
	default:
		return "", fmt.Errorf("unknown scope %q", scope)
	}
}

func Run(args ...string) (string, error) {
	cmd := exec.Command("launchctl", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("launchctl %s: %w\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String(), nil
}

func ServiceTarget(scope Scope, label string) (string, error) {
	domain, err := Domain(scope)
	if err != nil {
		return "", err
	}
	return domain + "/" + label, nil
}

func Loaded(scope Scope, label string) bool {
	target, err := ServiceTarget(scope, label)
	if err != nil {
		return false
	}
	_, err = Run("print", target)
	return err == nil
}
