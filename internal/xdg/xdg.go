package xdg

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func ConfigHome() (string, error) { return homeDir("XDG_CONFIG_HOME", ".config") }
func StateHome() (string, error)  { return homeDir("XDG_STATE_HOME", filepath.Join(".local", "state")) }
func CacheHome() (string, error)  { return homeDir("XDG_CACHE_HOME", ".cache") }
func DataHome() (string, error)   { return homeDir("XDG_DATA_HOME", filepath.Join(".local", "share")) }

func ConfigDirs() []string {
	value := os.Getenv("XDG_CONFIG_DIRS")
	if value == "" {
		return []string{"/etc/xdg"}
	}
	return absoluteList(strings.Split(value, string(os.PathListSeparator)))
}

func homeDir(envName, fallback string) (string, error) {
	if value := os.Getenv(envName); value != "" && filepath.IsAbs(value) {
		return value, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, fallback), nil
}

func absoluteList(paths []string) []string {
	var out []string
	for _, path := range paths {
		if filepath.IsAbs(path) {
			out = append(out, path)
		}
	}
	return out
}
