package cmd

import (
	"errors"
	"io"

	"github.com/Platon223/commitin/cli/internal/config"
	"github.com/Platon223/commitin/cli/internal/tui"
)

// requireLogin checks that a session token is saved locally, printing a
// styled message pointing to signup/login if not. This is a local-only
// check (no backend call) -- a token that's present locally but has expired
// or been revoked on the backend still passes here and fails later, when
// whatever command actually needs it makes its API call.
func requireLogin(out io.Writer, cfg *config.Config) error {
	if cfg.Token != "" {
		return nil
	}
	tui.PrintError(out, "not logged in — run `cmtin signup` to create an account, or `cmtin login` if you already have one")
	return errors.New("not logged in")
}
