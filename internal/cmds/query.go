// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/explain"
	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/query"
)

func newQueryCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "query",
		Short: "Run genealogical queries against the graph",
	}
	c.AddCommand(newQueryPathCmd(), newQueryNKCACmd())
	return c
}

func newQueryPathCmd() *cobra.Command {
	var (
		from, to       string
		maxPaths       int
		maxDepth       int
		includeAffinal bool
	)
	c := &cobra.Command{
		Use:   "path",
		Short: "Find ranked genealogical paths between two persons",
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
			fromID, err := model.ParseID(from)
			if err != nil {
				return err
			}
			toID, err := model.ParseID(to)
			if err != nil {
				return err
			}
			paths, err := query.FindPaths(ctx, rtx, fromID, toID, query.Options{
				MaxDepth:       maxDepth,
				MaxPaths:       maxPaths,
				IncludeAffinal: includeAffinal,
			})
			if err != nil {
				return err
			}
			out := make([]any, 0, len(paths))
			for _, p := range paths {
				exp, err := explain.Path(rtx, p)
				if err != nil {
					return err
				}
				out = append(out, exp)
			}
			result := map[string]any{
				"graphVersion": rtx.Version(),
				"paths":        out,
			}
			if current.Output == 1 { // OutputJSON
				return current.Print(os.Stdout, result)
			}
			// text mode: short summary
			for i, p := range paths {
				fmt.Fprintf(os.Stdout, "#%d  %s -> %s  len=%d  cert=%s  gap=%d/%d  %s\n",
					i+1, p.From(), p.To(), p.Length, p.Certainty, p.TotalGap, p.GapEdges, p.Classification)
			}
			fmt.Fprintf(os.Stdout, "(graph version %d)\n", rtx.Version())
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "source person id (required)")
	c.Flags().StringVar(&to, "to", "", "target person id (required)")
	c.Flags().IntVar(&maxPaths, "max-paths", 0, "limit ranked paths (0 = no limit)")
	c.Flags().IntVar(&maxDepth, "max-depth", 0, "limit traversal depth (0 = engine default)")
	c.Flags().BoolVar(&includeAffinal, "include-affinal", false, "allow Marriage edges in paths")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	return c
}

func newQueryNKCACmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nkca <person-a> <person-b>",
		Short: "Find the nearest known common ancestor of two persons",
		Args:  cobra.ExactArgs(2),
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
			a, err := model.ParseID(args[0])
			if err != nil {
				return err
			}
			b, err := model.ParseID(args[1])
			if err != nil {
				return err
			}
			res, err := query.NearestKnownCommonAncestor(ctx, rtx, a, b, query.Options{})
			if err != nil {
				return err
			}
			exp, err := explain.CommonAncestor(rtx, res)
			if err != nil {
				return err
			}
			if current.Output == 1 { // OutputJSON
				return current.Print(os.Stdout, exp)
			}
			fmt.Fprintf(os.Stdout, "ancestor: %s%s\n", exp.AncestorID,
				ifThen(exp.AncestorIsUnknown, " (unknown placeholder)", ""))
			fmt.Fprintf(os.Stdout, "total generations: %d\n", exp.TotalGenerations)
			fmt.Fprintf(os.Stdout, "combined certainty: %s\n", exp.CombinedCertainty)
			fmt.Fprintf(os.Stdout, "(graph version %d)\n", exp.GraphVersion)
			return nil
		},
	}
}

func ifThen(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}
