package cmd

import (
	"github.com/spf13/cobra"
)

// syncCmd governor resources
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync governor and okta resources",
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
