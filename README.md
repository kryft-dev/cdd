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
```

`cdd` takes no query: it always opens the Picker, and you filter by typing
into it. For a typed jump from the command line, use a tool like zoxide.
The Picker lists only Projects in History, and drops any whose `.git` has
since gone, along with any you have forgotten.

### Adding and forgetting Projects

A directory without a `.git` is not a Project until you add it:

```sh
cdd add ~/notes   # a Project from now on; it shows in the Picker at once
cdd add           # the current directory
cdd forget ~/old  # hide a Project for good, git repository or not
cdd forget        # the current directory
```

In the Picker, `ctrl+d` forgets the selected Project after a `y/n` confirm.

An added directory is listed even with no Visits (after the visited ones)
and drops out once the directory is gone. A forgotten Project is never
listed, and `cdd scan` does not bring it back. `cdd add` on a forgotten
Project undoes the forget. cdd records both in its own `projects` file and
never writes into your directories. See
[ADR 0002](docs/adr/0002-projects-can-be-added-by-hand.md).

### Query syntax

What you type into the Picker is split on spaces into words:


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
unless the query has an uppercase letter.

### Keys

Default key map:

| Key | Action |
| --- | --- |
| type | filter the list |
| `↑` / `ctrl+p` | move up |
| `↓` / `ctrl+n` | move down |
| `enter` | Jump to the selected Project (the `jump` Action) |
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
| `enter` | Jump to the selected Project (the `jump` Action) |
| `?` | open an overlay of every key, including your Actions' (list focus), and `?`, `esc` or `q` to close it |

### Key hints

The last line of the Picker lists the keys of your Actions, e.g.
`enter jump · ctrl+o files · ctrl+e editor · …`, with the match count at its
right edge. Whatever holds `enter` comes first, then your own `[actions.*]`
in the order `config.toml` declares them, then the built-ins; an Action with
`key = ""` is not listed. A line too narrow for all of them drops whole hints
from the end behind a `…`. A failure or confirmation (such as `copied`)
replaces the hints until the next key press.

With `keys.vim = true` the line also points to `?`, which opens an overlay of
every key, navigation included, and `?`, `esc` or `q` closes it. An Action
you bind to `?` takes the key from the overlay, and the default key map has
no overlay, since `?` is typing there. Set `[picker] hints = false` to hide
the hints, leaving the count and messages.

### Actions

An **Action** is a named command bound to a key and run on the selected
Project. Define one in `config.toml` (`cdd config init` writes a starting
file with examples):

```toml
[actions.code]
key    = "ctrl+v"
run    = "code {path}"   # {path} is the shell-quoted absolute path
detach = true            # start without waiting; the Picker stays open

[actions.lazygit]
key  = "ctrl+l"
run  = "lazygit"         # runs with the Project as its working directory
```

- `run` is handed to `sh -c` with the Project as the working directory and
  `$CDD_PATH` set to its path. Use `{path}`, not `$path`, which zsh ties to
  `$PATH`.
- By default the Picker quits and the command gets your terminal for stdin,
  stdout and stderr, so a TUI like `lazygit` works. `cdd` waits and exits
  with its status. Nothing is Jumped to afterwards, unless `jump = true`,
  which Jumps to the Project once the command exits successfully.
- `detach = true` starts the command in its own session without waiting,
  discarding its output, and the Picker stays open. If it cannot be
  started, the reason shows on the Picker's last line until the next key.
- Every run records a Visit to the Project.

Key names are those the Picker recognises: `ctrl+x`, `alt+x`, `enter`,
`f1`, and so on. A binding takes the key away from its navigation use
(`ctrl+n` bound means `↓` is the only way down), but `esc` and `ctrl+c`
can never be bound. A plain printable key (`a`, `?`) would steal typing, so
it is an error unless `keys.vim = true`, where it applies in list focus.
Two Actions you bind to one key is an error, though a built-in's default key
yields to yours, and `key = ""` leaves an Action unbound. Each of these is
reported with the line it is on.

An `[actions.<name>]` table whose name is a built-in Action overrides only
the fields it sets. The built-in Actions are:

| Action | Key | Runs |
| --- | --- | --- |
| `jump` | `enter` | Jumps to the Project, in both key maps and in the vim filter focus |
| `files` | `ctrl+o` | `xdg-open {path}` (`open {path}` on macOS), detached |
| `editor` | `ctrl+e` | `${VISUAL:-${EDITOR:-vi}} {path}`, on your terminal |
| `remote` | `ctrl+g` | `xdg-open {remote}` (`open {remote}` on macOS), detached |
| `copy` | `ctrl+y` | Copies the Project's path to the clipboard, and says "copied" on the last line |
| `forget` | `ctrl+d` | Forgets the Project, as `cdd forget` does, once you confirm |

`forget` asks `forget <name>? y/n` on the last line. `y` forgets the Project
and drops its row, leaving the cursor on the row that took its place; any
other key cancels, and is not typed into the filter. If cdd cannot write the
`projects` file, the reason shows in red and the row stays. `cdd add`
brings a forgotten Project back. Setting `run` or `jump` replaces it with an
ordinary command.

`copy` uses the first of `wl-copy` (when `WAYLAND_DISPLAY` is set), `xclip`,
`xsel` and `pbcopy` it finds on `PATH`. With none of them it sends the OSC 52
escape to the terminal instead, which also works over SSH in terminals that
allow it. Setting `run` replaces it with an ordinary command, such as
`run = "printf %s {path} | wl-copy"` with `detach = true`.

`editor` reads `$VISUAL`, else `$EDITOR`, else `vi`, each time it runs.
Override `run` to pick a program, or set `key = ""` to unbind any of them:

```toml
[actions.editor]
run = "hx {path}"        # replaces the built-in command

[actions.files]
key = ""                 # no file manager binding
```

`{remote}` in a `run` stands for the shell-quoted home page URL of the
Project's git remote: `origin`, else the first remote `git remote` lists,
rewritten to its repository page, never the branch.
`git@host:owner/repo.git` and `ssh://git@host:2222/owner/repo.git` both
become `https://host/owner/repo`, and an `https://` URL loses its `.git`;
GitLab subgroups (`owner/group/repo`) are kept. A Project with no remote, or
that is not a git repository, cannot run an Action that uses `{remote}`: the
Picker stays open and says so on its last line. Set `run` to use another
command or browser:

```toml
[actions.remote]
run = "firefox {remote}" # replaces the built-in command
```

Rebind `jump`, or give `enter` to another Action:

```toml
[actions.jump]
key = "alt+enter"        # Jump moves here

[actions.code]
key    = "enter"         # Enter now opens VS Code
run    = "code {path}"
detach = true
```

A key you set wins over a built-in that holds it by default: `jump` holds
`enter` until you give `enter` to another Action, which leaves `jump`
unbound, as if you had set `[actions.jump] key = ""`. So remapping `enter`
alone is enough; there is no need to move `jump` first. The same goes for
rebinding one built-in onto another built-in's key. Two Actions you bind to
the same key are still an error.

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
file is optional, and so is every key in it. `cdd config init` writes a
commented example with every key and some Action recipes (it refuses to
replace an existing file unless you pass `--force`), `cdd config path`
prints where the file lives, and `cdd config edit` opens it in `$VISUAL`,
else `$EDITOR`, else `vi`, running `init` first if there is none. The same
file ships as `config.example.toml` in each release archive. These are the
defaults:

```toml
exclude = []
include_hidden = false
[history]
max_visits = 1000
[keys]
vim = false
[picker]
layout = "list"
hints = true
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
- `[actions.<name>]`: an Action, with `key`, `run`, `jump` and `detach`;
  see [Actions](#actions). Not in the defaults above; `jump`, `files`,
  `editor`, `remote`, `copy` and `forget` exist unless you override them.
- `[picker].layout`: which Layout the Picker draws. `"list"`, described
  above, is the default and, for now, the only one. Any other value is a
  config error.
- `[picker].hints`: when `true` (the default), the last line lists the keys
  of your Actions; see [Key hints](#key-hints). `false` hides them.

Upgrading from v0.2: `root` and the `"grouped"` layout are gone. Delete
them from `config.toml` (cdd names the offending line), then run `cdd scan`
to rebuild History, since the old entries were stored relative to Root and
are ignored.

History is stored at `$XDG_DATA_HOME/cdd/history`, falling back to
`~/.local/share/cdd/history` when `XDG_DATA_HOME` is unset. The `projects`
file written by `cdd add` and `cdd forget` sits beside it. Each line is
`add <path>` or `forget <path>`, and the last line for a path wins.

## How it works

A Scan walks the directories it is given once and seeds History with a
Visit per git repository it finds, dated from the repository's last commit
(or the directory's mtime, before the first commit). The Picker lists
Projects drawn from History alone, most recently visited first, so opening
it never walks the disk; choosing a Project records a new Visit and hands
its path to the Wrapper, which turns it into a Jump in your shell. See
[ADR 0001](docs/adr/0001-projects-are-git-repos-found-by-scan.md) for why,
and [ADR 0002](docs/adr/0002-projects-can-be-added-by-hand.md) for the
directories you add by hand.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, the
[CONTEXT.md](CONTEXT.md) glossary, and the commit and pull request
workflow.

## License

[MIT](LICENSE)
