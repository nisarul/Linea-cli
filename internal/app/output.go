// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Print writes the given value to w in the App's selected output
// format. For text mode the value's String() is used if it
// implements fmt.Stringer; otherwise %v.
func (a *App) Print(w io.Writer, v any) error {
	if a.Output == OutputJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	switch t := v.(type) {
	case string:
		_, err := fmt.Fprintln(w, t)
		return err
	case fmt.Stringer:
		_, err := fmt.Fprintln(w, t.String())
		return err
	default:
		_, err := fmt.Fprintf(w, "%v\n", v)
		return err
	}
}

// PrintTable writes a simple aligned text table to w when the
// output mode is text; in JSON mode it encodes rows as objects
// keyed by the supplied headers.
func (a *App) PrintTable(w io.Writer, headers []string, rows [][]string) error {
	if a.Output == OutputJSON {
		out := make([]map[string]string, 0, len(rows))
		for _, r := range rows {
			obj := make(map[string]string, len(headers))
			for i, h := range headers {
				if i < len(r) {
					obj[h] = r[i]
				}
			}
			out = append(out, obj)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	writeRow := func(cells []string) {
		var b strings.Builder
		for i, c := range cells {
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(c)
			pad := widths[i] - len(c)
			for p := 0; p < pad; p++ {
				b.WriteByte(' ')
			}
		}
		fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	}
	writeRow(headers)
	sep := make([]string, len(headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	writeRow(sep)
	for _, r := range rows {
		writeRow(r)
	}
	return nil
}
