// SPDX-License-Identifier: AGPL-3.0-or-later

package cmds

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nisarul/Linea-core/model"
	"github.com/nisarul/Linea-core/store"
)

// JSONL line shape: {"kind": "person", "data": {...}} etc.
//
// The "kind" envelope keeps the format extensible without
// requiring a versioned manifest in v0.1.

type jsonlLine struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export the graph as JSONL to stdout (one entity per line)",
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
			out := bufio.NewWriter(os.Stdout)
			defer out.Flush()
			if err := emit(out, "version", map[string]any{"graphVersion": rtx.Version()}); err != nil {
				return err
			}
			err = rtx.IteratePersons(func(p model.Person) bool {
				_ = emit(out, "person", personJSON(p))
				return true
			})
			if err != nil {
				return err
			}
			return rtx.IterateRelationships(func(r model.Relationship) bool {
				_ = emit(out, "relationship", relationshipJSON(r))
				return true
			})
		},
	}
}

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import a JSONL graph dump from stdin (one entity per line)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			s, err := current.OpenStore(ctx)
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
			lineNo := 0
			_, err = s.Update(ctx, func(tx store.WriteTx) error {
				for scanner.Scan() {
					lineNo++
					line := strings.TrimSpace(scanner.Text())
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}
					var env jsonlLine
					if err := json.Unmarshal([]byte(line), &env); err != nil {
						return fmt.Errorf("line %d: %w", lineNo, err)
					}
					switch env.Kind {
					case "version":
						// purely informational
					case "person":
						p, err := decodePersonJSON(env.Data)
						if err != nil {
							return fmt.Errorf("line %d: %w", lineNo, err)
						}
						if err := tx.PutPerson(p); err != nil {
							return fmt.Errorf("line %d: %w", lineNo, err)
						}
					case "relationship":
						r, err := decodeRelationshipJSON(env.Data)
						if err != nil {
							return fmt.Errorf("line %d: %w", lineNo, err)
						}
						if err := tx.PutRelationship(r); err != nil {
							return fmt.Errorf("line %d: %w", lineNo, err)
						}
					default:
						return fmt.Errorf("line %d: unknown kind %q", lineNo, env.Kind)
					}
				}
				return scanner.Err()
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "imported %d lines\n", lineNo)
			return nil
		},
	}
}

// ----- emit/encode helpers -----

func emit(w io.Writer, kind string, data any) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	out := jsonlLine{Kind: kind, Data: buf}
	enc, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if _, err := w.Write(enc); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func personJSON(p model.Person) map[string]any {
	if p.IsUnknownAncestor() {
		return map[string]any{
			"id":      p.ID().String(),
			"unknown": true,
		}
	}
	return map[string]any{
		"id":     p.ID().String(),
		"names":  formatNames(p.Names()),
		"gender": string(p.Gender()),
		"notes":  p.Notes(),
	}
}

func relationshipJSON(r model.Relationship) map[string]any {
	cont := map[string]any{"state": r.Continuity().State.String()}
	if r.Continuity().IsGapped() {
		cont["gapKnown"] = r.Continuity().Gap.KnownSize
		if r.Continuity().Gap.KnownSize {
			cont["gapSize"] = r.Continuity().Gap.Size
		}
	}
	return map[string]any{
		"id":         r.ID().String(),
		"type":       r.Type().String(),
		"from":       r.From().String(),
		"to":         r.To().String(),
		"certainty":  r.Certainty().String(),
		"continuity": cont,
	}
}

// ----- decode helpers (mirror of personJSON / relationshipJSON) -----

type personImport struct {
	ID      string `json:"id"`
	Unknown bool   `json:"unknown,omitempty"`
	Names   []struct {
		Text      string `json:"text"`
		Language  string `json:"language,omitempty"`
		Script    string `json:"script,omitempty"`
		Type      string `json:"type,omitempty"`
		Preferred bool   `json:"preferred,omitempty"`
	} `json:"names,omitempty"`
	Gender string `json:"gender,omitempty"`
	Notes  string `json:"notes,omitempty"`
}

func decodePersonJSON(buf []byte) (model.Person, error) {
	var pi personImport
	if err := json.Unmarshal(buf, &pi); err != nil {
		return model.Person{}, err
	}
	id, err := model.ParseID(pi.ID)
	if err != nil {
		return model.Person{}, err
	}
	if pi.Unknown {
		return model.NewUnknownAncestor(id)
	}
	names := make([]model.Name, 0, len(pi.Names))
	for _, nn := range pi.Names {
		t := model.NameType(nn.Type)
		if t == "" {
			t = model.NameTypeFull
		}
		n, err := model.NewName(nn.Text, nn.Language, nn.Script, t, nn.Preferred)
		if err != nil {
			return model.Person{}, err
		}
		names = append(names, n)
	}
	g, err := model.ParseGender(pi.Gender, true)
	if err != nil {
		return model.Person{}, err
	}
	return model.NewPerson(id, model.PersonOptions{Names: names, Gender: g, Notes: pi.Notes})
}

type relationshipImport struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	From       string `json:"from"`
	To         string `json:"to"`
	Certainty  string `json:"certainty"`
	Continuity struct {
		State    string `json:"state"`
		GapKnown bool   `json:"gapKnown,omitempty"`
		GapSize  int    `json:"gapSize,omitempty"`
	} `json:"continuity"`
}

func decodeRelationshipJSON(buf []byte) (model.Relationship, error) {
	var ri relationshipImport
	if err := json.Unmarshal(buf, &ri); err != nil {
		return model.Relationship{}, err
	}
	id, err := model.ParseID(ri.ID)
	if err != nil {
		return model.Relationship{}, err
	}
	from, err := model.ParseID(ri.From)
	if err != nil {
		return model.Relationship{}, err
	}
	to, err := model.ParseID(ri.To)
	if err != nil {
		return model.Relationship{}, err
	}
	rt, err := parseRelType(ri.Type)
	if err != nil {
		return model.Relationship{}, err
	}
	c, err := parseCertainty(ri.Certainty)
	if err != nil {
		return model.Relationship{}, err
	}
	var cont model.Continuity
	switch ri.Continuity.State {
	case "Continuous":
		cont = model.NewContinuous()
	case "Gapped":
		if ri.Continuity.GapKnown {
			gg, err := model.KnownGap(ri.Continuity.GapSize)
			if err != nil {
				return model.Relationship{}, err
			}
			cont = model.NewGapped(gg)
		} else {
			cont = model.NewGapped(model.UnknownGap())
		}
	default:
		return model.Relationship{}, fmt.Errorf("unknown continuity state %q", ri.Continuity.State)
	}
	return model.NewRelationship(id, from, to, rt, c, cont, model.RelationshipOptions{})
}
