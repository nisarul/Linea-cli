// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/store"
)

func newSourceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "source",
		Short: "Manage source citations",
	}
	c.AddCommand(newSourceAddCmd(), newSourceListCmd())
	return c
}

func newSourceAddCmd() *cobra.Command {
	var (
		typ      string
		citation string
		author   string
		title    string
		date     string
		locator  string
	)
	c := &cobra.Command{
		Use:   "add",
		Short: "Create a source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			src, err := model.NewSource(model.NewID(), model.SourceType(typ), citation,
				model.SourceOptions{Author: author, Title: title, Date: date, Locator: locator})
			if err != nil {
				return err
			}
			_, err = s.Update(ctx, func(tx store.WriteTx) error { return tx.PutSource(src) })
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, src.ID())
			return nil
		},
	}
	c.Flags().StringVar(&typ, "type", "other", "source type: primary|secondary|oral|derived|other")
	c.Flags().StringVar(&citation, "citation", "", "human-readable citation text (required)")
	c.Flags().StringVar(&author, "author", "", "author")
	c.Flags().StringVar(&title, "title", "", "title")
	c.Flags().StringVar(&date, "date", "", "publication / origin date")
	c.Flags().StringVar(&locator, "locator", "", "page / folio / URL")
	_ = c.MarkFlagRequired("citation")
	return c
}

func newSourceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			rtx, err := s.View(ctx)
			if err != nil {
				return err
			}
			defer rtx.Close()
			var rows [][]string
			err = rtx.IterateSources(func(src model.Source) bool {
				rows = append(rows, []string{src.ID().String(), string(src.Type()), src.Citation()})
				return true
			})
			if err != nil {
				return err
			}
			return current.PrintTable(os.Stdout, []string{"id", "type", "citation"}, rows)
		},
	}
}
