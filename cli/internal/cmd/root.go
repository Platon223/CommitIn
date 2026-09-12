// Package cmd defines the cmtin CLI's commands.
package cmd

import (
	"os"

	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/spf13/cobra"
)

// apiURL holds the --api-url flag / COMMITIN_API_URL env value. Empty means
// "not explicitly set" — resolveAPIURL then falls back to the saved config,
// then a hardcoded default.
var apiURL string

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "cmtin",
		Short:        "CommitIn — judges your git commit messages",
		SilenceUsage: true,
	}
	root.PersistentFlags().StringVar(&apiURL, "api-url", envDefault("COMMITIN_API_URL", ""),
		"CommitIn backend base URL (env COMMITIN_API_URL; falls back to the saved session, then http://localhost:8080)")
	root.AddCommand(newSignupCmd())
	root.AddCommand(newLoginCmd())
	root.AddCommand(newLogoutCmd())
	return root
}

// Execute runs the CLI. Errors are printed by cobra; the caller should exit
// non-zero when it returns a non-nil error.
func Execute() error {
	return newRootCmd().Execute()
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resolveAPIURL picks the backend URL: --api-url / COMMITIN_API_URL first,
// then the URL saved in cfg from a previous login, then a hardcoded default.
func resolveAPIURL(cfg *config.Config) string {
	if apiURL != "" {
		return apiURL
	}
	if cfg != nil && cfg.APIURL != "" {
		return cfg.APIURL
	}
	return "http://localhost:8080"
}
