// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	linea "github.com/nisarul/Linea-core"

	"github.com/nisarul/Linea-cli/internal/buildinfo"
)

// newInitCmd creates an empty Linea data directory by opening
// the Badger store once (which seeds the version key) and
// closing it. This makes `linea init` an explicit "claim" of
// the directory.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise a Linea data directory at --data-dir",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			v, err := s.CurrentVersion(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "initialised %s (version %d, spec %s)\n",
				current.DataDir, v, linea.SpecVersion)
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the linea CLI version and the Linea spec version it implements",
		Run: func(_ *cobra.Command, _ []string) {
			b := buildinfo.Get()
			fmt.Printf("linea %s (commit %s, built %s) — linea-core spec %s\n",
				b.Version, b.Commit, b.Date, linea.SpecVersion)
		},
	}
}
