# Linea CLI

Command-line tool for [Linea](https://github.com/nisarul/Linea-specs) — a
domain-neutral, truth-aware genealogical graph framework.

`linea` is the dogfooding consumer of [`Linea-core`](https://github.com/nisarul/Linea-core).
It stores a graph in an embedded Badger database, lets you propose changes
through the governance state machine, and runs spec-conformant queries
against the result.

> Linea — lineage, without assumptions.

## Status

Pre-release. Implements spec v1.1.0 via Linea-core.

## Install

```sh
go install github.com/nisarul/Linea-cli/cmd/linea@latest
```

## Quick start

```sh
linea init
linea person add --name "Alice"
linea person add --name "Bob"
# ...add a relationship...
linea proposal submit <id>
linea proposal accept <id>
linea query path --from <a-id> --to <b-id>
```

## Configuration

| Flag         | Env var          | Default              |
|--------------|------------------|----------------------|
| `--data-dir` | `LINEA_DATA_DIR` | `~/.linea/data`      |
| `--actor`    | `LINEA_ACTOR`    | OS user name         |
| `--output`   | `LINEA_OUTPUT`   | `text` (or `json`)   |

## Exit codes

| Code | Meaning                                        |
|-----:|------------------------------------------------|
|  `0` | Success                                        |
|  `1` | Error (validation, governance, IO, etc.)       |
|  `2` | `NO_KNOWN_CONNECTION` (queries that found none) |

This lets scripts distinguish *"no path exists"* from *"the command broke"*.

## License

AGPL-3.0-or-later. See [LICENSE](./LICENSE).
The Linea specifications themselves are licensed under CC BY 4.0.
