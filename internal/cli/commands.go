package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	return compileUnitPath(path, unitName, runAtLoad)
}

func installUnitPath(scope launchd.Scope, path, unitName string, runAtLoad bool) error {
	job, err := compileUnitPath(path, unitName, runAtLoad)
	if err != nil {
		return err
	}
	return installJob(scope, job)
}

func compileUnitPath(path, unitName string, runAtLoad bool) (launchd.Job, error) {
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

func label(name string) string {
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".service"), ".timer")
	if strings.HasPrefix(name, "dev.macctl.") {
		return name
	}
	return "dev.macctl." + name
}
