// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/store"
)

func newPersonCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "person",
		Short: "Manage persons in the Linea graph",
	}
	c.AddCommand(newPersonAddCmd(), newPersonListCmd(), newPersonShowCmd())
	return c
}

func newPersonAddCmd() *cobra.Command {
	var (
		name    string
		gender  string
		notes   string
		unknown bool
	)
	c := &cobra.Command{
		Use:   "add",
		Short: "Create a person directly (does NOT go through proposals)",
		Long: "Adds a Person to the graph. This bypasses the proposal pipeline " +
			"and is intended for bootstrapping / ingest. For governed changes use the " +
			"`linea proposal *` family of commands.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			id := model.NewID()
			var person model.Person
			if unknown {
				if name != "" || gender != "" || notes != "" {
					return fmt.Errorf("unknown-ancestor placeholders may not carry name/gender/notes (CCGGS 5.3)")
				}
				person, err = model.NewUnknownAncestor(id)
				if err != nil {
					return err
				}
			} else {
				if name == "" {
					return fmt.Errorf("--name is required (use --unknown for an unknown-ancestor placeholder)")
				}
				n, err := model.NewName(name, "", "", model.NameTypeFull, true)
				if err != nil {
					return err
				}
				g, err := model.ParseGender(gender, true)
				if err != nil {
					return err
				}
				person, err = model.NewPerson(id, model.PersonOptions{
					Names:  []model.Name{n},
					Gender: g,
					Notes:  notes,
				})
				if err != nil {
					return err
				}
			}
			_, err = s.Update(ctx, func(tx store.WriteTx) error { return tx.PutPerson(person) })
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, person.ID())
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "preferred name (required unless --unknown)")
	c.Flags().StringVar(&gender, "gender", "", "gender (male|female|unknown or extended vocabulary)")
	c.Flags().StringVar(&notes, "notes", "", "free-form notes")
	c.Flags().BoolVar(&unknown, "unknown", false, "create an unknown-ancestor placeholder (CCGGS 5.3)")
	return c
}

func newPersonListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all persons in the graph",
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
			err = rtx.IteratePersons(func(p model.Person) bool {
				name := p.PreferredName().Text
				kind := "person"
				if p.IsUnknownAncestor() {
					name = ""
					kind = "unknown-ancestor"
				}
				rows = append(rows, []string{p.ID().String(), kind, name})
				return true
			})
			if err != nil {
				return err
			}
			return current.PrintTable(os.Stdout, []string{"id", "kind", "name"}, rows)
		},
	}
}

func newPersonShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a single person by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			id, err := model.ParseID(args[0])
			if err != nil {
				return err
			}
			p, err := rtx.GetPerson(id)
			if err != nil {
				return err
			}
			out := map[string]any{
				"id":              p.ID().String(),
				"unknownAncestor": p.IsUnknownAncestor(),
				"gender":          string(p.Gender()),
				"notes":           p.Notes(),
				"names":           formatNames(p.Names()),
				"graphVersion":    rtx.Version(),
			}
			return current.Print(os.Stdout, out)
		},
	}
}

func formatNames(ns []model.Name) []map[string]any {
	out := make([]map[string]any, 0, len(ns))
	for _, n := range ns {
		out = append(out, map[string]any{
			"text":      n.Text,
			"language":  n.Language,
			"script":    n.Script,
			"type":      string(n.Type),
			"preferred": n.Preferred,
		})
	}
	return out
}
