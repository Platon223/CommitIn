// Package cmd defines the cmtin CLI's commands.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var apiURL string

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "cmtin",
		Short:         "CommitIn — judges your git commit messages",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().StringVar(&apiURL, "api-url", envDefault("COMMITIN_API_URL", "http://localhost:8080"), "CommitIn backend base URL")
	root.AddCommand(newSignupCmd())
	root.AddCommand(newLoginCmd())
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
