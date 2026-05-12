// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/store"
)

func newRelationshipCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "relationship",
		Aliases: []string{"rel"},
		Short:   "Manage relationships between persons",
	}
	c.AddCommand(newRelationshipAddCmd(), newRelationshipListCmd())
	return c
}

func newRelationshipAddCmd() *cobra.Command {
	var (
		from       string
		to         string
		typ        string
		certainty  string
		gappedSize int
		gappedUnk  bool
	)
	c := &cobra.Command{
		Use:   "add",
		Short: "Create a relationship directly (use proposals for governed changes)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			fromID, err := model.ParseID(from)
			if err != nil {
				return err
			}
			toID, err := model.ParseID(to)
			if err != nil {
				return err
			}
			rt, err := parseRelType(typ)
			if err != nil {
				return err
			}
			c, err := parseCertainty(certainty)
			if err != nil {
				return err
			}
			cont := model.NewContinuous()
			switch {
			case gappedUnk:
				cont = model.NewGapped(model.UnknownGap())
			case gappedSize > 0:
				gg, err := model.KnownGap(gappedSize)
				if err != nil {
					return err
				}
				cont = model.NewGapped(gg)
			}
			r, err := model.NewRelationship(model.NewID(), fromID, toID, rt, c, cont,
				model.RelationshipOptions{})
			if err != nil {
				return err
			}
			_, err = s.Update(ctx, func(tx store.WriteTx) error { return tx.PutRelationship(r) })
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, r.ID())
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "source person id (required)")
	c.Flags().StringVar(&to, "to", "", "target person id (required)")
	c.Flags().StringVar(&typ, "type", "ParentChild", "relationship type: ParentChild|Marriage")
	c.Flags().StringVar(&certainty, "certainty", "Certain", "certainty: Certain|Probable|Uncertain")
	c.Flags().IntVar(&gappedSize, "gap-size", 0, "treat as Gapped with N intermediate generations (>0)")
	c.Flags().BoolVar(&gappedUnk, "gap-unknown", false, "treat as Gapped with unknown size")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	return c
}

func newRelationshipListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all relationships in the graph",
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
			err = rtx.IterateRelationships(func(r model.Relationship) bool {
				rows = append(rows, []string{
					r.ID().String(), r.Type().String(),
					r.From().String(), r.To().String(),
					r.Certainty().String(), r.Continuity().String(),
				})
				return true
			})
			if err != nil {
				return err
			}
			return current.PrintTable(os.Stdout,
				[]string{"id", "type", "from", "to", "certainty", "continuity"}, rows)
		},
	}
}

func parseRelType(s string) (model.RelationshipType, error) {
	switch s {
	case "ParentChild":
		return model.RelTypeParentChild, nil
	case "Marriage":
		return model.RelTypeMarriage, nil
	}
	return 0, fmt.Errorf("unknown relationship type %q (use ParentChild|Marriage)", s)
}

func parseCertainty(s string) (model.Certainty, error) {
	switch s {
	case "Certain":
		return model.CertaintyCertain, nil
	case "Probable":
		return model.CertaintyProbable, nil
	case "Uncertain":
		return model.CertaintyUncertain, nil
	}
	return 0, fmt.Errorf("unknown certainty %q (use Certain|Probable|Uncertain)", s)
}
