// SPDX-License-Identifier: AGPL-3.0-or-later

// Package buildinfo carries the linker-injected version metadata
// for the linea CLI binary so commands can read it without having
// to import package main.
package buildinfo

// Info is the build-time metadata bundle.
type Info struct {
	Version string
	Commit  string
	Date    string
}

var current Info

// Set is called once from main during process startup.
func Set(i Info) { current = i }

// Get returns the current build-time metadata.
func Get() Info { return current }
