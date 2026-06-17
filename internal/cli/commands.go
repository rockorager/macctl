package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	envd "go.rockorager.dev/macctl/internal/env"
	"go.rockorager.dev/macctl/internal/launchd"
	unitd "go.rockorager.dev/macctl/internal/unit"
)

func startCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "start UNIT...", ValidArgsFunction: completeStartUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("start", args); err != nil {
			return err
		}
		for _, name := range args {
			if unitd.LooksLikePathOrUnitFile(name) {
				job, err := compileUnit(scope(opts), name, false)
				if err != nil {
					return err
				}
				if err := installJob(scope(opts), job); err != nil {
					return err
				}
			}
			target, err := launchd.ServiceTarget(scope(opts), label(name))
			if err != nil {
				return err
			}
			if out, err := launchd.Run("kickstart", "-k", target); err != nil {
				return err
			} else if out != "" {
				fmt.Print(out)
			}
		}
		return nil
	}}
}

func stopCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "stop UNIT...", ValidArgsFunction: completeGeneratedUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("stop", args); err != nil {
			return err
		}
		for _, name := range args {
			target, err := launchd.ServiceTarget(scope(opts), label(name))
			if err != nil {
				return err
			}
			if out, err := launchd.Run("bootout", target); err != nil {
				return err
			} else if out != "" {
				fmt.Print(out)
			}
		}
		return nil
	}}
}

func restartCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "restart UNIT...", ValidArgsFunction: completeGeneratedUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("restart", args); err != nil {
			return err
		}
		for _, name := range args {
			target, err := launchd.ServiceTarget(scope(opts), label(name))
			if err != nil {
				return err
			}
			if out, err := launchd.Run("kickstart", "-k", target); err != nil {
				return err
			} else if out != "" {
				fmt.Print(out)
			}
		}
		return nil
	}}
}

func statusCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "status UNIT...", ValidArgsFunction: completeGeneratedUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("status", args); err != nil {
			return err
		}
		for i, name := range args {
			if i > 0 {
				fmt.Println()
			}
			if err := printStatus(scope(opts), name); err != nil {
				return err
			}
		}
		return nil
	}}
}

func enableCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "enable UNIT...", ValidArgsFunction: completeConfigUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("enable", args); err != nil {
			return err
		}
		for _, arg := range args {
			if err := installUnit(scope(opts), arg); err != nil {
				return err
			}
		}
		return nil
	}}
}

func disableCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "disable UNIT...", ValidArgsFunction: completeEnabledUnits(opts), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("disable", args); err != nil {
			return err
		}
		for _, name := range args {
			target, err := launchd.ServiceTarget(scope(opts), label(name))
			if err != nil {
				return err
			}
			if out, err := launchd.Run("disable", target); err != nil {
				return err
			} else if out != "" {
				fmt.Print(out)
			}
		}
		return nil
	}}
}

func listUnitFilesCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "list-unit-files", RunE: func(cmd *cobra.Command, args []string) error {
		files, err := unitd.UnitFiles(scope(opts))
		if err != nil {
			return err
		}
		fmt.Printf("%-32s %s\n", "UNIT FILE", "STATE")
		for _, file := range files {
			fmt.Printf("%-32s %s\n", file.Name, file.State)
		}
		return nil
	}}
}

func daemonReloadCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "daemon-reload", RunE: func(cmd *cobra.Command, args []string) error {
		vars, err := envd.Load(scope(opts))
		if err != nil {
			return err
		}
		if err := envd.Apply(scope(opts), vars); err != nil {
			return err
		}
		if scope(opts) == launchd.ScopeUser {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if err := installJob(scope(opts), envd.LaunchdJob(exe)); err != nil {
				return err
			}
		}
		paths, err := unitd.UnitPaths(scope(opts))
		if err != nil {
			return err
		}
		for _, path := range paths {
			if err := installUnitPath(scope(opts), path, filepath.Base(path), true); err != nil {
				return err
			}
		}
		return nil
	}}
}

func setEnvironmentCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "set-environment NAME=VALUE...", RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("set-environment", args); err != nil {
			return err
		}
		for _, assignment := range args {
			key, value, ok := strings.Cut(assignment, "=")
			if !ok || key == "" {
				return fmt.Errorf("expected NAME=VALUE, got %q", assignment)
			}
			if _, err := launchd.Run("setenv", key, value); err != nil {
				return err
			}
		}
		return nil
	}}
}

func unsetEnvironmentCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "unset-environment NAME...", RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("unset-environment", args); err != nil {
			return err
		}
		for _, key := range args {
			if _, err := launchd.Run("unsetenv", key); err != nil {
				return err
			}
		}
		return nil
	}}
}

func showEnvironmentCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "show-environment", RunE: func(cmd *cobra.Command, args []string) error {
		vars, err := envd.Load(scope(opts))
		if err != nil {
			return err
		}
		keys := make([]string, 0, len(vars))
		for key := range vars {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("%s=%s\n", key, vars[key])
		}
		return nil
	}}
}

func importEnvironmentCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "import-environment NAME...", RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireArgs("import-environment", args); err != nil {
			return err
		}
		for _, key := range args {
			if value, ok := os.LookupEnv(key); ok {
				if _, err := launchd.Run("setenv", key, value); err != nil {
					return err
				}
			}
		}
		return nil
	}}
}

func installUnit(scope launchd.Scope, name string) error {
	job, err := compileUnit(scope, name, true)
	if err != nil {
		return err
	}
	return installJob(scope, job)
}

func compileUnit(scope launchd.Scope, name string, runAtLoad bool) (launchd.Job, error) {
	path, err := unitd.ResolvePath(scope, name)
	if err != nil {
		return launchd.Job{}, err
	}
	unitName := filepath.Base(name)
	if strings.ContainsRune(name, os.PathSeparator) {
		unitName = filepath.Base(path)
	}
	return compileUnitPath(scope, path, unitName, runAtLoad)
}

func installUnitPath(scope launchd.Scope, path, unitName string, runAtLoad bool) error {
	job, err := compileUnitPath(scope, path, unitName, runAtLoad)
	if err != nil {
		return err
	}
	return installJob(scope, job)
}

func compileUnitPath(scope launchd.Scope, path, unitName string, runAtLoad bool) (launchd.Job, error) {
	var job launchd.Job
	switch {
	case strings.HasSuffix(unitName, ".service"):
		svc, err := unitd.LoadServiceAs(path, unitName)
		if err != nil {
			return launchd.Job{}, err
		}
		job = svc.LaunchdJob()
	case strings.HasSuffix(unitName, ".timer"):
		timer, err := unitd.LoadTimerAs(path, unitName)
		if err != nil {
			return launchd.Job{}, err
		}
		job = timer.LaunchdJob()
	default:
		return launchd.Job{}, fmt.Errorf("unsupported unit type %q", unitName)
	}
	if runAtLoad {
		job.RunAtLoad = true
	}
	applyDefaultLogPaths(scope, &job)
	return job, nil
}

func installJob(scope launchd.Scope, job launchd.Job) error {
	dir, err := launchd.PlistDir(scope)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := createLogDirs(job); err != nil {
		return err
	}
	plistPath := filepath.Join(dir, job.Label+".plist")
	changed, err := launchd.WritePlistIfChanged(plistPath, job)
	if err != nil {
		return err
	}
	if !changed && launchd.Loaded(scope, job.Label) {
		return nil
	}
	domain, err := launchd.Domain(scope)
	if err != nil {
		return err
	}
	if changed && launchd.Loaded(scope, job.Label) {
		target, targetErr := launchd.ServiceTarget(scope, job.Label)
		if targetErr != nil {
			return targetErr
		}
		_, _ = launchd.Run("bootout", target)
	}
	if out, err := launchd.Run("bootstrap", domain, plistPath); err != nil {
		return err
	} else if out != "" {
		fmt.Print(out)
	}
	fmt.Printf("enabled %s\n", job.Label)
	return nil
}

func applyDefaultLogPaths(scope launchd.Scope, job *launchd.Job) {
	if len(job.ProgramArguments) == 0 || (job.StandardOutPath != "" && job.StandardErrorPath != "") {
		return
	}
	path := defaultLogPath(scope, strings.TrimPrefix(job.Label, "dev.macctl."))
	if job.StandardOutPath == "" {
		job.StandardOutPath = path
	}
	if job.StandardErrorPath == "" {
		job.StandardErrorPath = path
	}
}

func defaultLogPath(scope launchd.Scope, name string) string {
	if scope == launchd.ScopeSystem {
		return filepath.Join("/var/log/macctl", name+".log")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Library", "Logs", "macctl", name+".log")
	}
	return filepath.Join(os.TempDir(), "macctl", name+".log")
}

func createLogDirs(job launchd.Job) error {
	seen := map[string]bool{}
	for _, path := range []string{job.StandardOutPath, job.StandardErrorPath} {
		if path == "" || path == "/dev/null" || seen[path] {
			continue
		}
		seen[path] = true
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func label(name string) string {
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".service"), ".timer")
	if strings.HasPrefix(name, "dev.macctl.") {
		return name
	}
	return "dev.macctl." + name
}

func printStatus(scope launchd.Scope, name string) error {
	unitName, err := statusUnitName(scope, name)
	if err != nil {
		return err
	}
	label := label(unitName)
	unitPath, unitErr := unitd.ResolvePath(scope, unitName)
	description := ""
	if unitErr == nil && strings.HasSuffix(unitName, ".service") {
		if svc, err := unitd.LoadServiceAs(unitPath, unitName); err == nil {
			description = svc.Description
		}
	}
	if description == "" {
		description = unitName
	}

	plistPath, job, err := generatedJob(scope, label)
	if err != nil {
		return err
	}
	state := launchdState{}
	if out, err := launchd.Run("print", mustServiceTarget(scope, label)); err == nil {
		state = parseLaunchdState(out)
	}
	loaded := "loaded"
	if !launchd.Loaded(scope, label) {
		loaded = "not-found"
	}
	enabled := "disabled"
	if job.RunAtLoad {
		enabled = "enabled"
	}

	fmt.Printf("● %s - %s\n", unitName, description)
	if unitErr == nil {
		fmt.Printf("     Loaded: %s (%s; %s)\n", loaded, unitPath, enabled)
	} else {
		fmt.Printf("     Loaded: %s (%s; %s)\n", loaded, plistPath, enabled)
	}
	fmt.Printf("     Active: %s\n", activeLine(state))
	if state.PID != "" {
		fmt.Printf("   Main PID: %s\n", state.PID)
	}
	if state.LastExitCode != "" {
		fmt.Printf("  Exit Code: %s\n", state.LastExitCode)
	}
	fmt.Printf("      Label: %s\n", label)
	fmt.Printf("      Plist: %s\n", plistPath)
	if logPath := logPathLine(job); logPath != "" {
		fmt.Printf("       Logs: %s\n", logPath)
	}
	logs := recentLogs(state, job)
	if len(logs) > 0 {
		fmt.Println()
		for _, line := range logs {
			fmt.Println(line)
		}
	}
	return nil
}

func logPathLine(job launchd.Job) string {
	if job.StandardOutPath == "" && job.StandardErrorPath == "" {
		return ""
	}
	if job.StandardOutPath != "" && job.StandardOutPath == job.StandardErrorPath {
		return job.StandardOutPath
	}
	var parts []string
	if job.StandardOutPath != "" {
		parts = append(parts, "stdout: "+job.StandardOutPath)
	}
	if job.StandardErrorPath != "" {
		parts = append(parts, "stderr: "+job.StandardErrorPath)
	}
	return strings.Join(parts, "; ")
}

func statusUnitName(scope launchd.Scope, name string) (string, error) {
	generated, err := unitd.GeneratedUnitNames(scope)
	if err != nil {
		return "", err
	}
	want := unitd.LabelName(name)
	if path, err := unitd.ResolvePath(scope, name); err == nil && generatedHasLabel(generated, filepath.Base(path)) {
		return filepath.Base(path), nil
	}
	for _, unitName := range generated {
		if unitName == name || unitd.LabelName(unitName) == want {
			for _, candidate := range []string{unitName, unitName + ".service", unitName + ".timer"} {
				if path, err := unitd.ResolvePath(scope, candidate); err == nil {
					return filepath.Base(path), nil
				}
			}
			return unitName, nil
		}
	}
	return "", fmt.Errorf("unit %q is not installed by macctl", name)
}

func generatedHasLabel(generated []string, name string) bool {
	want := unitd.LabelName(name)
	for _, unitName := range generated {
		if unitd.LabelName(unitName) == want {
			return true
		}
	}
	return false
}

func generatedJob(scope launchd.Scope, label string) (string, launchd.Job, error) {
	dir, err := launchd.PlistDir(scope)
	if err != nil {
		return "", launchd.Job{}, err
	}
	path := filepath.Join(dir, label+".plist")
	job, err := launchd.ReadPlist(path)
	if err != nil {
		return "", launchd.Job{}, err
	}
	return path, job, nil
}

func mustServiceTarget(scope launchd.Scope, label string) string {
	target, err := launchd.ServiceTarget(scope, label)
	if err != nil {
		return label
	}
	return target
}

type launchdState struct {
	State        string
	PID          string
	LastExitCode string
}

func parseLaunchdState(out string) launchdState {
	state := launchdState{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, " = ")
		if !ok {
			continue
		}
		switch key {
		case "state":
			if state.State == "" {
				state.State = value
			}
		case "pid":
			state.PID = value
		case "last exit code":
			state.LastExitCode = value
		}
	}
	return state
}

func activeLine(state launchdState) string {
	switch state.State {
	case "running":
		return "active (running)"
	case "exited":
		return "inactive (exited)"
	case "waiting":
		return "inactive (waiting)"
	case "not running":
		return "inactive (dead)"
	case "":
		return "inactive (not loaded)"
	default:
		return state.State
	}
}

func recentLogs(state launchdState, job launchd.Job) []string {
	if lines := recentFileLogs(job); len(lines) > 0 {
		return lines
	}
	args := []string{"show", "--style", "compact", "--last", "10m"}
	if state.PID != "" {
		args = append(args, "--predicate", fmt.Sprintf("processID == %s", state.PID))
	} else if len(job.ProgramArguments) > 0 {
		args = append(args, "--process", filepath.Base(job.ProgramArguments[0]))
	} else {
		return nil
	}
	cmd := exec.Command("log", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := compactLogLines(string(out))
	if len(lines) > 10 {
		lines = lines[len(lines)-10:]
	}
	return lines
}

func recentFileLogs(job launchd.Job) []string {
	seen := map[string]bool{}
	var paths []string
	for _, path := range []string{job.StandardOutPath, job.StandardErrorPath} {
		if path == "" || path == "/dev/null" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	var lines []string
	for _, path := range paths {
		pathLines := tailLines(path, 10)
		if len(pathLines) == 0 {
			continue
		}
		if len(paths) > 1 {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, "==> "+path+" <==")
		}
		lines = append(lines, pathLines...)
	}
	return lines
}

func tailLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			copy(lines, lines[1:])
			lines = lines[:n]
		}
	}
	if scanner.Err() != nil {
		return nil
	}
	return lines
}

var logTimestamp = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2}) (\d{2}:\d{2}:\d{2})(?:\.\d+)?(?:[-+]\d{4})? `)

func compactLogLines(out string) []string {
	now := time.Now()
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Timestamp") || strings.HasPrefix(line, "Filtering") || strings.HasPrefix(line, "Skipping") {
			continue
		}
		line = logTimestamp.ReplaceAllStringFunc(line, func(ts string) string {
			parts := logTimestamp.FindStringSubmatch(ts)
			if len(parts) != 5 {
				return ts
			}
			parsed, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s-%s-%s %s", parts[1], parts[2], parts[3], parts[4]))
			if err != nil {
				return ts
			}
			if parsed.Year() == now.Year() {
				return parsed.Format("Jan _2 15:04:05 ")
			}
			return parsed.Format("Jan _2  2006 ")
		})
		lines = append(lines, line)
	}
	return lines
}
