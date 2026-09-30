#!/usr/bin/env bash
# Builds a self-contained demo environment for recording cdd, so the
# recording never touches your real home, config, or History.
#
#   demo/setup.sh            # builds /tmp/cdd-demo
#   DEMO_DIR=~/x demo/setup.sh
#
# Layout it creates:
#   $DEMO_DIR/home         a fake home with git repositories at several depths
#   $DEMO_DIR/xdg/config   XDG_CONFIG_HOME holding cdd/config.toml
#   $DEMO_DIR/xdg/data     XDG_DATA_HOME holding cdd/history
#   $DEMO_DIR/bin/cdd      the binary built from this checkout
set -euo pipefail

DEMO_DIR="${DEMO_DIR:-/tmp/cdd-demo}"
ROOT="$DEMO_DIR/home"
CONFIG="$DEMO_DIR/xdg/config"
DATA="$DEMO_DIR/xdg/data"
BIN="$DEMO_DIR/bin"
REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"

rm -rf "$DEMO_DIR"
mkdir -p "$ROOT" "$CONFIG/cdd" "$DATA/cdd" "$BIN"

# --- Projects -----------------------------------------------------------
# Each Project is a git repository unless noted. Commit dates are spread
# out so `cdd scan` seeds a varied History.
git_demo() { git -c user.name=demo -c user.email=demo@example.com -c commit.gpgsign=false "$@"; }

repo() { # repo <path under home> <days-ago> <file>
  local dir="$ROOT/$1"; mkdir -p "$dir"
  git_demo -C "$dir" init -q -b main
  echo "# ${1##*/}" > "$dir/$3"
  local when; when="$(ago "$2")"
  GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" git_demo -C "$dir" -c init.defaultBranch=main add -A
  GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" git_demo -C "$dir" commit -q -m "initial commit"
}

ago() { # ago <days> -> ISO timestamp, GNU or BSD date
  date -u -d "$1 days ago" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
    || date -u -v-"$1"d +%Y-%m-%dT%H:%M:%SZ
}

repo Developer/domain/kryft.dev          1  README.md
repo Developer/domain/wowaitech.com      3  README.md
repo Developer/tools/cdd                 0  main.go
repo Developer/tools/grg                 6  main.go
repo Developer/tools/lab/bubbletea-play  5  main.go
repo work/internal                       2  README.md
repo work/clients/acme/api               9  README.md
repo dotfiles                           40  README.md
repo uni/2025/compilers                 20  notes.md

# clean vs. dirty vs. untracked vs. ahead
echo "wip" >> "$ROOT/Developer/domain/wowaitech.com/README.md"      # modified
touch "$ROOT/Developer/tools/grg/scratch.go"                        # untracked
echo "wip" >> "$ROOT/work/internal/README.md"; touch "$ROOT/work/internal/new.txt"   # both

upstream="$DEMO_DIR/upstream.git"
git_demo init -q --bare "$upstream"
git_demo -C "$ROOT/Developer/domain/kryft.dev" remote add origin "$upstream"
git_demo -C "$ROOT/Developer/domain/kryft.dev" push -q -u origin main
echo "more" >> "$ROOT/Developer/domain/kryft.dev/README.md"
git_demo -C "$ROOT/Developer/domain/kryft.dev" commit -q -am "second commit"   # ahead by 1

mkdir -p "$ROOT/notes/2026"                                    # not a repository, never listed
repo Developer/tools/cdd/vendor/inner 1 README.md              # inside a repository, never listed
repo .cache/some-tool 1 README.md                              # hidden, never listed

# --- Config and History ---------------------------------------------------
# No config.toml: cdd needs none.

# --- Binary and seeded History --------------------------------------------
( cd "$REPO_DIR" && go build -o "$BIN/cdd" ./cmd/cdd )
HOME="$ROOT" XDG_CONFIG_HOME="$CONFIG" XDG_DATA_HOME="$DATA" "$BIN/cdd" scan

# A few real Jumps on top of the Scan so the ordering shows both sources.
hist="$DATA/cdd/history"
printf '%s\tjump\t%s\n' "$(ago 0)" "$ROOT/Developer/tools/cdd"        >> "$hist"
printf '%s\tjump\t%s\n' "$(ago 1)" "$ROOT/Developer/domain/kryft.dev" >> "$hist"
printf '%s\tjump\t%s\n' "$(ago 2)" "$ROOT/work/internal"              >> "$hist"

cat <<MSG

Demo environment ready in $DEMO_DIR

Try it in a shell:
  set -gx XDG_CONFIG_HOME $CONFIG; set -gx XDG_DATA_HOME $DATA   # fish
  export XDG_CONFIG_HOME=$CONFIG XDG_DATA_HOME=$DATA               # bash/zsh
  HOME=$ROOT $BIN/cdd

Record it (VHS v0.11.0; v0.12.0 silently writes no GIF):
  vhs demo/demo.tape
MSG
