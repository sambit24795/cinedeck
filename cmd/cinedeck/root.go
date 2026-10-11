package main

import "github.com/spf13/cobra"

var version = "dev"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "cinedeck",
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().Bool("json", false, "output as JSON")
	cmd.PersistentFlags().String("region", "", "region code")
	cmd.PersistentFlags().String("config", "", "config file path")
	_ = cmd.PersistentFlags().MarkHidden("config")
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newInfoCmd())
	cmd.AddCommand(newShortsCmd())
	cmd.AddCommand(newTrendingCmd())
	cmd.AddCommand(newWhereCmd())
	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newKdramaCmd())
	cmd.AddCommand(newAnimeCmd())
	cmd.AddCommand(newLikeCmd())
	cmd.AddCommand(newUnlikeCmd())
	cmd.AddCommand(newRateCmd())
	cmd.AddCommand(newUnrateCmd())
	cmd.AddCommand(newWatchedCmd())
	cmd.AddCommand(newUnwatchCmd())
	return cmd
}
