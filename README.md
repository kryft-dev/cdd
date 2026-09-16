# cdd

A TUI that lets you jump to recent Projects.

## Demo

<!-- Maintainer: record a terminal-session GIF or asciinema cast of the
     Picker in action and drop it here. No recording tool is available in
     this environment. -->

## Install

### GitHub Releases

Download a prebuilt binary from the [Releases](https://github.com/kryft-dev/cdd/releases)
page. Archives are published as `tar.gz` for `linux` and `darwin`, each in
`amd64` and `arm64`. Verify a download against the release's
`checksums.txt`:

```sh
sha256sum --check --ignore-missing checksums.txt
```

Then extract the archive and put the `cdd` binary on your `PATH`.

### go install

```sh
go install github.com/kryft-dev/cdd/cmd/cdd@latest
```

## Shell setup

`cdd` needs a shell Wrapper so choosing a Project in the Picker can actually
Jump your shell there — a subprocess cannot change its parent shell's
working directory on its own.

**fish** — add to `~/.config/fish/config.fish`:

```fish
cdd init fish | source
```

**bash** — add to `~/.bashrc`:

```sh
eval "$(cdd init bash)"
```

**zsh** — add to `~/.zshrc`:

```sh
eval "$(cdd init zsh)"
```

## First run

Create a `config.toml` (see [Config reference](#config-reference)) with at
least a `root`, then seed History with a Scan of everything already under
Root:

```sh
cdd scan
```

## Usage

```sh
cdd            # open the Picker over History, ordered by recency
cdd <query>    # open the Picker pre-filtered by query
```

### Keys

Default key map:

| Key | Action |
| --- | --- |
| type | filter the list |
| `↑` / `ctrl+p` | move up |
| `↓` / `ctrl+n` | move down |
| `enter` | Jump to the selected Project |
| `esc` / `ctrl+c` | cancel |
| `ctrl+u` | clear the filter |

Vim key map (`keys.vim = true`): the list is focused on open.

| Key | Action |
| --- | --- |
| `j` | move down |
| `k` | move up |
| `g` / `G` | jump to first / last row |
| `f` / `/` | focus the filter |
| `esc` (filter) | return to the list, keeping the query |
| `esc` / `q` | cancel |
| `enter` | Jump to the selected Project |

## Config reference

`cdd` reads `config.toml` from `$XDG_CONFIG_HOME/cdd/config.toml`, falling
back to `~/.config/cdd/config.toml` when `XDG_CONFIG_HOME` is unset.

```toml
root = "~/Developer"      # required, no default
exclude = []              # paths relative to Root, glob-matched
include_hidden = false
[history]
max_visits = 1000         # must be >= 1
[keys]
vim = false
```

- `root` (required): the top-level directory whose Kinds are searched for
  Projects. A leading `~` is expanded to the user's home directory;
  `$VAR` is left as-is.
- `exclude`: paths relative to Root, matched with `path.Match` semantics,
  that are skipped when discovering Kinds and Projects.
- `include_hidden`: when `false` (the default), hidden directories are
  excluded at both Kind and Project level.
- `[history].max_visits`: the maximum number of Visits kept in History.
  Must be at least 1; defaults to `1000`.
- `[keys].vim`: when `true`, the Picker opens with the list focused and
  uses the vim key map described above. Defaults to `false`, the default
  key map.

History is stored at `$XDG_DATA_HOME/cdd/history`, falling back to
`~/.local/share/cdd/history` when `XDG_DATA_HOME` is unset.

## How it works

A Scan sweeps Root once and seeds History with a Visit per discovered
Project, dated from evidence found inside the Project itself. The Picker
lists Projects drawn from History, most recently visited first, and lets
you choose one; choosing a Project records a new Visit and hands its path
to the Wrapper, which turns it into a Jump in your shell.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the
[CONTEXT.md](CONTEXT.md) glossary, and the commit and pull request
workflow.

## License

[MIT](LICENSE)
