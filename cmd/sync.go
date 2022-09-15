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

	syncCmd.PersistentFlags().Bool("dry-run", false, "do not make any changes when running a sync")
	viperBindFlag("sync.dryrun", syncCmd.PersistentFlags().Lookup("dry-run"))

	// Okta related flags
	syncCmd.PersistentFlags().String("okta-url", "https://equinixmetal.okta.com", "url for Okta client calls")
	viperBindFlag("okta.url", syncCmd.PersistentFlags().Lookup("okta-url"))
	syncCmd.PersistentFlags().String("okta-token", "", "token for access to the Okta API")
	viperBindFlag("okta.token", syncCmd.PersistentFlags().Lookup("okta-token"))
	syncCmd.PersistentFlags().Bool("okta-nocache", false, "disable the okta client cache, useful for development")
	viperBindFlag("okta.nocache", syncCmd.PersistentFlags().Lookup("okta-nocache"))

	// Governor related flags
	syncCmd.PersistentFlags().String("governor-url", "https://api.governor.metalkube.net", "url of the governor api")
	viperBindFlag("governor.url", syncCmd.PersistentFlags().Lookup("governor-url"))
	syncCmd.PersistentFlags().String("governor-client-id", "gov-okta-addon-governor", "oauth client ID for client credentials flow")
	viperBindFlag("governor.client-id", syncCmd.PersistentFlags().Lookup("governor-client-id"))
	syncCmd.PersistentFlags().String("governor-client-secret", "", "oauth client secret for client credentials flow")
	viperBindFlag("governor.client-secret", syncCmd.PersistentFlags().Lookup("governor-client-secret"))
	syncCmd.PersistentFlags().String("governor-token-url", "http://hydra:4444/oauth2/token", "url used for client credential flow")
	viperBindFlag("governor.token-url", syncCmd.PersistentFlags().Lookup("governor-token-url"))
	syncCmd.PersistentFlags().String("governor-audience", "https://api.governor.metalkube.net", "oauth audience for client credential flow")
	viperBindFlag("governor.audience", syncCmd.PersistentFlags().Lookup("governor-audience"))
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}

	return false
}
