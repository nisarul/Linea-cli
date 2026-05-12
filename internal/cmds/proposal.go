// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/governance"
	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/store"
)

func newProposalCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "proposal",
		Aliases: []string{"prop"},
		Short:   "Manage governance proposals",
	}
	c.AddCommand(
		newProposalCreatePersonCmd(),
		newProposalCreateRelCmd(),
		newProposalListCmd(),
		newProposalShowCmd(),
		newProposalSubmitCmd(),
		newProposalClaimCmd(),
		newProposalAcceptCmd(),
		newProposalRejectCmd(),
		newProposalWithdrawCmd(),
	)
	return c
}

// ---- create-person proposal ----

func newProposalCreatePersonCmd() *cobra.Command {
	var (
		name    string
		gender  string
		notes   string
		unknown bool
		reason  string
	)
	c := &cobra.Command{
		Use:   "create-person",
		Short: "Submit a Draft proposal that creates a Person",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			payload := governance.PayloadCreatePerson{
				UnknownAncestor: unknown,
				Gender:          model.Gender(gender),
				Notes:           notes,
			}
			if !unknown {
				if name == "" {
					return fmt.Errorf("--name is required (use --unknown for an unknown-ancestor placeholder)")
				}
				n, err := model.NewName(name, "", "", model.NameTypeFull, true)
				if err != nil {
					return err
				}
				payload.Names = []model.Name{n}
			}
			plBuf, _ := json.Marshal(payload)
			pp, err := model.NewProposal(model.NewID(), model.ProposalActionCreate,
				model.EntityKindPerson, model.ProposalOptions{
					Payload: plBuf, Reason: reason, Author: current.Actor,
					CreatedAt: time.Now().Unix(),
				})
			if err != nil {
				return err
			}
			_, err = s.Update(ctx, func(tx store.WriteTx) error { return tx.PutProposal(pp) })
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, pp.ID())
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "preferred name (required unless --unknown)")
	c.Flags().StringVar(&gender, "gender", "", "gender")
	c.Flags().StringVar(&notes, "notes", "", "notes")
	c.Flags().BoolVar(&unknown, "unknown", false, "create an unknown-ancestor placeholder")
	c.Flags().StringVar(&reason, "reason", "", "rationale recorded on the proposal")
	return c
}

// ---- create-relationship proposal ----

func newProposalCreateRelCmd() *cobra.Command {
	var (
		from, to, typ, certainty, reason string
		gappedSize                       int
		gappedUnk                        bool
	)
	c := &cobra.Command{
		Use:   "create-relationship",
		Short: "Submit a Draft proposal that creates a Relationship",
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
			cval, err := parseCertainty(certainty)
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
			payload := governance.PayloadCreateRelationship{
				From: fromID, To: toID, Type: rt,
				Certainty: cval, Continuity: cont,
			}
			plBuf, _ := json.Marshal(payload)
			pp, err := model.NewProposal(model.NewID(), model.ProposalActionCreate,
				model.EntityKindRelationship, model.ProposalOptions{
					Payload: plBuf, Reason: reason, Author: current.Actor,
					CreatedAt: time.Now().Unix(),
				})
			if err != nil {
				return err
			}
			_, err = s.Update(ctx, func(tx store.WriteTx) error { return tx.PutProposal(pp) })
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, pp.ID())
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "source person id (required)")
	c.Flags().StringVar(&to, "to", "", "target person id (required)")
	c.Flags().StringVar(&typ, "type", "ParentChild", "type: ParentChild|Marriage")
	c.Flags().StringVar(&certainty, "certainty", "Certain", "certainty: Certain|Probable|Uncertain")
	c.Flags().IntVar(&gappedSize, "gap-size", 0, "Gapped continuity with N intermediate generations")
	c.Flags().BoolVar(&gappedUnk, "gap-unknown", false, "Gapped continuity with unknown size")
	c.Flags().StringVar(&reason, "reason", "", "rationale recorded on the proposal")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	return c
}

// ---- listing / showing ----

func newProposalListCmd() *cobra.Command {
	var stateFilter string
	c := &cobra.Command{
		Use:   "list",
		Short: "List proposals (optionally filter by --state)",
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
			err = rtx.IterateProposals(func(p model.Proposal) bool {
				if stateFilter != "" && p.State().String() != stateFilter {
					return true
				}
				rows = append(rows, []string{
					p.ID().String(), p.State().String(),
					p.Action().String(), p.EntityKind().String(),
					p.Author(),
				})
				return true
			})
			if err != nil {
				return err
			}
			return current.PrintTable(os.Stdout,
				[]string{"id", "state", "action", "kind", "author"}, rows)
		},
	}
	c.Flags().StringVar(&stateFilter, "state", "",
		"only list proposals in this state (Draft|Submitted|UnderReview|Accepted|Rejected|Withdrawn)")
	return c
}

func newProposalShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a single proposal and its history",
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
			p, err := rtx.GetProposal(id)
			if err != nil {
				return err
			}
			out := map[string]any{
				"id":          p.ID().String(),
				"state":       p.State().String(),
				"action":      p.Action().String(),
				"entityKind":  p.EntityKind().String(),
				"targetId":    p.TargetID().String(),
				"secondaryId": p.SecondaryID().String(),
				"reason":      p.Reason(),
				"author":      p.Author(),
				"history":     formatHistory(p.History()),
				"graphVersion": rtx.Version(),
			}
			return current.Print(os.Stdout, out)
		},
	}
}

func formatHistory(hs []model.ProposalTransition) []map[string]any {
	out := make([]map[string]any, 0, len(hs))
	for _, h := range hs {
		out = append(out, map[string]any{
			"from":      h.From.String(),
			"to":        h.To.String(),
			"actor":     h.Actor,
			"timestamp": h.Timestamp,
			"reason":    h.Reason,
		})
	}
	return out
}

// ---- transition commands ----

func newProposalSubmitCmd() *cobra.Command {
	return transitionCmd("submit", "Move a Draft proposal to Submitted",
		func(ctx contextLike, s store.Store, id model.ID, reason string) error {
			_, err := governance.Submit(ctx.Ctx(), s, id, current.Actor, time.Now().Unix())
			return err
		}, false)
}

func newProposalClaimCmd() *cobra.Command {
	return transitionCmd("claim", "Move a Submitted proposal to UnderReview",
		func(ctx contextLike, s store.Store, id model.ID, reason string) error {
			_, err := governance.Claim(ctx.Ctx(), s, id, current.Actor, time.Now().Unix())
			return err
		}, false)
}

func newProposalAcceptCmd() *cobra.Command {
	return transitionCmd("accept", "Accept an UnderReview proposal (applies its mutation)",
		func(ctx contextLike, s store.Store, id model.ID, reason string) error {
			_, err := governance.Accept(ctx.Ctx(), s, id, current.Actor, time.Now().Unix())
			return err
		}, false)
}

func newProposalRejectCmd() *cobra.Command {
	return transitionCmd("reject", "Reject an UnderReview proposal (--reason required)",
		func(ctx contextLike, s store.Store, id model.ID, reason string) error {
			_, err := governance.Reject(ctx.Ctx(), s, id, current.Actor, time.Now().Unix(), reason)
			return err
		}, true)
}

func newProposalWithdrawCmd() *cobra.Command {
	return transitionCmd("withdraw", "Withdraw a non-terminal proposal",
		func(ctx contextLike, s store.Store, id model.ID, reason string) error {
			_, err := governance.Withdraw(ctx.Ctx(), s, id, current.Actor, time.Now().Unix(), reason)
			return err
		}, false)
}

// transitionCmd builds a one-shot proposal-state transition cobra command.
func transitionCmd(
	use, short string,
	apply func(contextLike, store.Store, model.ID, string) error,
	requireReason bool,
) *cobra.Command {
	var reason string
	c := &cobra.Command{
		Use:   use + " <proposal-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			id, err := model.ParseID(args[0])
			if err != nil {
				return err
			}
			if requireReason && reason == "" {
				return fmt.Errorf("--reason is required for this transition")
			}
			return apply(realCtx{ctx}, s, id, reason)
		},
	}
	c.Flags().StringVar(&reason, "reason", "", "rationale (required for reject)")
	return c
}

// contextLike is a tiny abstraction so transitionCmd's callbacks
// don't import context directly into their closures and stay compact.
type contextLike interface{ Ctx() ctxStdLib }

type realCtx struct{ c ctxStdLib }

func (r realCtx) Ctx() ctxStdLib { return r.c }

// ctxStdLib aliases context.Context to avoid importing it twice.
type ctxStdLib = interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
