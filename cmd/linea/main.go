// SPDX-License-Identifier: AGPL-3.0-or-later

// Command linea is the Linea CLI. See https://github.com/nisarul/Linea-cli.
package main

import (
	"fmt"
	"os"

	lerrors "github.com/nisarul/Linea-core/errors"

	"github.com/nisarul/Linea-cli/internal/cmds"
)

func main() {
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
