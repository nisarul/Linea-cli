// SPDX-License-Identifier: AGPL-3.0-or-later

// Command linea is the Linea CLI. See https://github.com/nisarul/Linea-cli.
package main

import (
	"fmt"
	"os"

	lerrors "github.com/nisarul/Linea-core/errors"

	"github.com/nisarul/Linea-cli/internal/buildinfo"
	"github.com/nisarul/Linea-cli/internal/cmds"
)

// These are populated at build time via -ldflags. Defaults are
// suitable for `go run` / `go build` invocations.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	buildinfo.Set(buildinfo.Info{Version: version, Commit: commit, Date: date})

	if err := cmds.NewRoot().Execute(); err != nil {
		// Distinguish NO_KNOWN_CONNECTION (exit 2) from other errors.
		if lerrors.IsNoKnownConnection(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
