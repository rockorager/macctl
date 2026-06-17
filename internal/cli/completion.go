package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"go.rockorager.dev/macctl/internal/launchd"
	unitd "go.rockorager.dev/macctl/internal/unit"
)

func completeConfigUnits(opts *options) cobra.CompletionFunc {
	return completeFrom(func() ([]string, error) { return unitd.ConfigUnitNames(scope(opts)) })
}

func completeGeneratedUnits(opts *options) cobra.CompletionFunc {
	return completeFrom(func() ([]string, error) { return unitd.GeneratedUnitNames(scope(opts)) })
}

func completeEnabledUnits(opts *options) cobra.CompletionFunc {
	return completeFrom(func() ([]string, error) { return unitd.EnabledUnitNames(scope(opts)) })
}

func completeStartUnits(opts *options) cobra.CompletionFunc {
	return completeFrom(func() ([]string, error) {
		config, err := unitd.ConfigUnitNames(scope(opts))
		if err != nil {
			return nil, err
		}
		generated, err := unitd.GeneratedUnitNames(scope(opts))
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		var names []string
		for _, name := range append(config, generated...) {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		return names, nil
	})
}

func completeFrom(load func() ([]string, error)) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, err := load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		used := map[string]bool{}
		for _, arg := range args {
			used[arg] = true
			used[unitd.LabelName(arg)] = true
		}
		var completions []string
		for _, name := range names {
			if used[name] || used[unitd.LabelName(name)] {
				continue
			}
			if strings.HasPrefix(name, toComplete) || strings.HasPrefix(unitd.LabelName(name), toComplete) {
				completions = append(completions, name)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func scope(opts *options) launchd.Scope { return launchd.Scope(opts.scope) }
