package main

import (
	"strings"

	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/spf13/cobra"
)

// regionFor returns the region to use: the --region flag when set, else
// cfg.Region. The result is trimmed and upper-cased; it may be "" if both are
// empty.
func regionFor(cmd *cobra.Command, cfg config.Config) string {
	region, _ := cmd.Root().PersistentFlags().GetString("region")
	if strings.TrimSpace(region) == "" {
		region = cfg.Region
	}
	return strings.ToUpper(strings.TrimSpace(region))
}
