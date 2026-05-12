// SPDX-License-Identifier: AGPL-3.0-or-later

// Package cmds wires together every cobra command exposed by the
// linea binary. NewRoot is the only exported entry point.
package cmds

import (
	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-cli/internal/app"
)

// shared global flags. Resolved into the App at PersistentPreRunE.
var (
	flagDataDir string
	flagActor   string
	flagOutput  string

	current = &app.App{}
)

// NewRoot returns the top-level cobra command tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "linea",
		Short:         "Linea — lineage, without assumptions.",
		Long:          "Command-line client for the Linea genealogical graph framework.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&flagDataDir, "data-dir", "",
		"path to the Linea data directory (env LINEA_DATA_DIR; default ~/.linea/data)")
	root.PersistentFlags().StringVar(&flagActor, "actor", "",
		"opaque actor identifier recorded on proposal transitions (env LINEA_ACTOR)")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "",
		"output format: text|json (env LINEA_OUTPUT; default text)")

	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		dir, err := app.ResolveDataDir(flagDataDir)
		if err != nil {
			return err
		}
		current.DataDir = dir
		current.Actor = app.ResolveActor(flagActor)
		current.Output = app.ResolveOutput(flagOutput)
		return nil
	}
	root.PersistentPostRun = func(_ *cobra.Command, _ []string) {
		_ = current.CloseStore()
	}

	root.AddCommand(
		newInitCmd(),
		newVersionCmd(),
		newPersonCmd(),
		newRelationshipCmd(),
		newSourceCmd(),
		newProposalCmd(),
		newQueryCmd(),
		newExportCmd(),
		newImportCmd(),
	)
	return root
}
