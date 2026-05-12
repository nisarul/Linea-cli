// SPDX-License-Identifier: AGPL-3.0-or-later

// Package app holds CLI-wide concerns: data-dir resolution, store
// open/close, and shared formatting helpers. Commands consume App
// values; they should not import store/badger or filesystem
// helpers directly.
package app

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/nisarul/Linea-core/store"
	"github.com/nisarul/Linea-core/store/badger"
)

// App carries CLI-wide state: configured paths, identity,
// output mode. It is constructed once per command invocation
// and threaded through to every command handler.
type App struct {
	DataDir string
	Actor   string
	Output  OutputMode

	store *badger.Store
}

// OutputMode controls how Print* helpers serialise results.
type OutputMode int

const (
	// OutputText prints human-readable, table-ish output.
	OutputText OutputMode = iota
	// OutputJSON prints a single JSON object per command.
	OutputJSON
)

// ResolveDataDir returns the data directory the CLI will use.
//
// Resolution order: explicit non-empty arg > LINEA_DATA_DIR env >
// "~/.linea/data". The directory is NOT created here.
func ResolveDataDir(flag string) (string, error) {
	if flag != "" {
		return filepath.Clean(flag), nil
	}
	if env := strings.TrimSpace(os.Getenv("LINEA_DATA_DIR")); env != "" {
		return filepath.Clean(env), nil
	}
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(u.HomeDir, ".linea", "data"), nil
}

// ResolveActor returns the actor identity used to record proposal
// transitions. Order: explicit > LINEA_ACTOR env > OS user.
func ResolveActor(flag string) string {
	if flag != "" {
		return flag
	}
	if env := strings.TrimSpace(os.Getenv("LINEA_ACTOR")); env != "" {
		return env
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}

// ResolveOutput returns the output mode for this invocation.
// Order: explicit flag > LINEA_OUTPUT env > OutputText.
// Unknown values fall back to OutputText with no error so the CLI
// stays scriptable; callers may inspect the original raw string
// via ParseOutput if they need stricter validation.
func ResolveOutput(flag string) OutputMode {
	v := flag
	if v == "" {
		v = os.Getenv("LINEA_OUTPUT")
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "json":
		return OutputJSON
	default:
		return OutputText
	}
}

// OpenStore opens (creating directory if needed) the Badger
// store at App.DataDir. It is safe to call once per command.
//
// Callers MUST defer App.CloseStore() in a top-level handler.
func (a *App) OpenStore(_ context.Context) (store.Store, error) {
	if a.store != nil {
		return a.store, nil
	}
	if err := os.MkdirAll(a.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", a.DataDir, err)
	}
	s, err := badger.Open(a.DataDir, badger.Silent())
	if err != nil {
		return nil, fmt.Errorf("open store at %s: %w", a.DataDir, err)
	}
	a.store = s
	return s, nil
}

// CloseStore releases the underlying database.
func (a *App) CloseStore() error {
	if a.store == nil {
		return nil
	}
	err := a.store.Close()
	a.store = nil
	return err
}
