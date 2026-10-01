# cdd

A TUI that lets you jump to the git repositories you work in, most recent
first. There is nothing to lay out or configure: `cdd scan` finds every
repository below the directories you give it, at any depth.

## Demo

<!-- Recorded with VHS against a throwaway home: demo/setup.sh && vhs demo/demo.tape -->
![cdd demo](demo/demo.gif)

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

Seed History with a Scan. With no arguments it walks your home directory;
give it one or more directories to walk those instead:

```sh
cdd scan                       # every git repository below ~
cdd scan ~/Developer ~/work    # only below these
```

A Project is any directory holding a `.git` directory or file (so
worktrees and submodules count). The walk goes to any depth but stops at a
repository, so repositories nested inside one are not listed, unless you
name that repository to `cdd scan` itself. It does not follow symlinks,
skips hidden directories and anything matched by `exclude`, and passes over
directories it cannot read. Run it again whenever you clone something new:
it never overrides the date of a real Jump.

## Usage

```sh
cdd            # open the Picker over History, ordered by recency
cdd <query>    # Jump straight there when exactly one Project matches, else
               # open the Picker pre-filtered by query
cdd baz br     # open the Picker on Projects named like "br" under a "baz" parent
```

A query matches a Project straight away when it is the Project's name
(`cdd cdd`), a trailing part of its path (`cdd tools/cdd`), or its whole
path. The Picker lists only Projects in History, and drops any whose `.git`
has since gone.

### Query syntax

| Query | Matches |
| --- | --- |
| `barbar` | the whole shown path, parent directory then name |
| `baz br` | name like `br`, under a parent directory like `baz` |
| `kryft tools cdd` | name like `cdd`, parent words in order (`~/kryft/tools/`) |
| `baz ` | any Project under a parent like `baz` (trailing space) |
| ` br` | names like `br` only (leading space) |
| ` ` | everything, as if empty |

Each word matches fuzzily, in order (`br` finds `barbar`), and survives a
typo: one edit for words of 4–7 letters, two from 8 (`brabar` still finds
`barbar`). Words of up to 3 letters must be exact. Typo matches rank below
every exact one; equally good matches keep History order. Case is ignored
unless the query has an uppercase letter. A query with a space never takes
the straight-to-Jump shortcut.

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

### Layout

The Picker draws the **List Layout**: a flat fzf-style run of Projects in
History order, each Project's parent directory muted before its name (`~`
standing in for your home directory, and trimmed from the start on a narrow
terminal), the filter prompt below the list, and a `▌` bar plus a
background highlight on the selected row. The filter matches the parent
directory as well as the name (see [Query syntax](#query-syntax)).

## Config reference

`cdd` reads `config.toml` from `$XDG_CONFIG_HOME/cdd/config.toml`, falling
back to `~/.config/cdd/config.toml` when `XDG_CONFIG_HOME` is unset. The
file is optional, and so is every key in it; these are the defaults:

```toml
exclude = []
include_hidden = false
[history]
max_visits = 1000
[keys]
vim = false
[picker]
layout = "list"
```

- `exclude`: glob patterns (`filepath.Match` semantics) for directories
  `cdd scan` skips, along with everything below them. A pattern containing
  `/` is matched against the directory's absolute path, any other against
  its name alone, as in `.gitignore`: `["node_modules", "~/go/pkg/*"]`. A
  leading `~` is expanded to your home directory; `$VAR` is left as-is.
- `include_hidden`: when `false` (the default), `cdd scan` skips hidden
  directories.
- `[history].max_visits`: the maximum number of Visits kept in History.
  Must be at least 1; defaults to `1000`.
- `[keys].vim`: when `true`, the Picker opens with the list focused and
  uses the vim key map described above. Defaults to `false`, the default
  key map.
- `[picker].layout`: which Layout the Picker draws. `"list"`, described
  above, is the default and, for now, the only one. Any other value is a
  config error.

Upgrading from v0.2: `root` and the `"grouped"` layout are gone. Delete
them from `config.toml` (cdd names the offending line), then run `cdd scan`
to rebuild History, since the old entries were stored relative to Root and
are ignored.

History is stored at `$XDG_DATA_HOME/cdd/history`, falling back to
`~/.local/share/cdd/history` when `XDG_DATA_HOME` is unset.

## How it works

A Scan walks the directories it is given once and seeds History with a
Visit per git repository it finds, dated from the repository's last commit
(or the directory's mtime, before the first commit). The Picker lists
Projects drawn from History alone, most recently visited first, so opening
it never walks the disk; choosing a Project records a new Visit and hands
its path to the Wrapper, which turns it into a Jump in your shell. See
[ADR 0001](docs/adr/0001-projects-are-git-repos-found-by-scan.md) for why.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the
[CONTEXT.md](CONTEXT.md) glossary, and the commit and pull request
workflow.

## License

[MIT](LICENSE)
