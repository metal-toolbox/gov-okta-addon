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

	// Okta related flags
	syncCmd.Flags().String("okta-url", "https://equinixmetal.okta.com", "url for Okta client calls")
	viperBindFlag("okta.url", syncCmd.Flags().Lookup("okta-url"))
	syncCmd.Flags().String("okta-token", "", "token for access to the Okta API")
	viperBindFlag("okta.token", syncCmd.Flags().Lookup("okta-token"))
	syncCmd.Flags().Bool("okta-nocache", false, "disable the okta client cache, useful for development")
	viperBindFlag("okta.nocache", syncCmd.Flags().Lookup("okta-nocache"))

	// Governor related flags
	syncCmd.Flags().String("governor-url", "https://api.governor.metalkube.net", "url of the governor api")
	viperBindFlag("governor.url", syncCmd.Flags().Lookup("governor-url"))
	syncCmd.Flags().String("governor-client-id", "gov-okta-addon-governor", "oauth client ID for client credentials flow")
	viperBindFlag("governor.client-id", syncCmd.Flags().Lookup("governor-client-id"))
	syncCmd.Flags().String("governor-client-secret", "", "oauth client secret for client credentials flow")
	viperBindFlag("governor.client-secret", syncCmd.Flags().Lookup("governor-client-secret"))
	syncCmd.Flags().String("governor-token-url", "http://hydra:4444/oauth2/token", "url used for client credential flow")
	viperBindFlag("governor.token-url", syncCmd.Flags().Lookup("governor-token-url"))
	syncCmd.Flags().String("governor-audience", "https://api.governor.metalkube.net", "oauth audience for client credential flow")
	viperBindFlag("governor.audience", syncCmd.Flags().Lookup("governor-audience"))
}
