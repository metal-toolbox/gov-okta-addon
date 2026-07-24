// Package cmd is our cobra/viper cli implementation
package cmd

import (
	"strings"

	sdkcfg "github.com/metal-toolbox/governor-extension-sdk/pkg/configs"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/metal-toolbox/gov-okta-addon/internal/configs"
)

const (
	appName = "gov-okta-addon"
)

var (
	cfgFile string
	logger  *zap.SugaredLogger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   appName,
	Short: "Integrates Governor and Okta",
	Long:  `gov-okta-addon is a microservice that integrates group/user management in governor with Okta.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	v := viper.GetViper()
	flags := rootCmd.PersistentFlags()

	flags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/."+appName+".yaml)")

	sdkcfg.MustLoggingFlags(v, flags)
	sdkcfg.MustTracingFlags(v, flags)
	sdkcfg.MustAuditFlags(v, flags)
	sdkcfg.MustGovernorFlags(v, flags)
	configs.MustOktaFlags(v, flags)
	configs.MustAddonFlags(v, flags)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gov-okta-addon" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("." + appName)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetEnvPrefix("goa")
	viper.AutomaticEnv() // read in environment variables that match

	setupLogging()

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		logger.Infow("using config file",
			"file", viper.ConfigFileUsed(),
		)
	}

	if err := viper.Unmarshal(&configs.AppConfig); err != nil {
		logger.Fatalw("failed to unmarshal config", "error", err)
	}
}

func setupLogging() {
	cfg := zap.NewProductionConfig()
	if viper.GetBool("logging.pretty") {
		cfg = zap.NewDevelopmentConfig()
	}

	if viper.GetBool("logging.debug") {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger = l.Sugar().With("app", appName)
	defer logger.Sync() //nolint:errcheck
}
