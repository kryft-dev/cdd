#!/usr/bin/env bash
# Builds a self-contained demo environment for recording cdd, so the
# recording never touches your real Root, config, or History.
#
#   demo/setup.sh            # builds /tmp/cdd-demo
#   DEMO_DIR=~/x demo/setup.sh
#
# Layout it creates:
#   $DEMO_DIR/root         a fake Root with five Kinds and a few Projects
#   $DEMO_DIR/xdg/config   XDG_CONFIG_HOME holding cdd/config.toml
#   $DEMO_DIR/xdg/data     XDG_DATA_HOME holding cdd/history
#   $DEMO_DIR/bin/cdd      the binary built from this checkout
set -euo pipefail

DEMO_DIR="${DEMO_DIR:-/tmp/cdd-demo}"
ROOT="$DEMO_DIR/root"
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

repo() { # repo <kind/name> <days-ago> <file>
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

repo domain/kryft.dev        1  README.md
repo domain/wowaitech.com    3  README.md
repo domain/candidex.dev    12  README.md
repo tools/cdd               0  main.go
repo tools/grg               6  main.go
repo tools/dotfiles         40  README.md
repo ops/internal            2  README.md
repo ops/backups            90  README.md
repo lab/bubbletea-play      5  main.go
repo uni/compilers          20  notes.md

# clean vs. dirty vs. untracked vs. ahead
echo "wip" >> "$ROOT/domain/wowaitech.com/README.md"          # modified
touch "$ROOT/tools/grg/scratch.go"                              # untracked
echo "wip" >> "$ROOT/ops/internal/README.md"; touch "$ROOT/ops/internal/new.txt"   # both

upstream="$DEMO_DIR/upstream.git"
git_demo init -q --bare "$upstream"
git_demo -C "$ROOT/domain/kryft.dev" remote add origin "$upstream"
git_demo -C "$ROOT/domain/kryft.dev" push -q -u origin main
echo "more" >> "$ROOT/domain/kryft.dev/README.md"
git_demo -C "$ROOT/domain/kryft.dev" commit -q -am "second commit"   # ahead by 1

mkdir -p "$ROOT/lab/notes"                                       # not a repository
mkdir -p "$ROOT/.archive/old-thing" "$ROOT/uni/.hidden"          # hidden, never listed

# --- Config and History ---------------------------------------------------
cat > "$CONFIG/cdd/config.toml" <<TOML
root = "$ROOT"
exclude = []
include_hidden = false

[history]
max_visits = 1000

[keys]
vim = false
TOML

# --- Binary and seeded History --------------------------------------------
( cd "$REPO_DIR" && go build -o "$BIN/cdd" ./cmd/cdd )
XDG_CONFIG_HOME="$CONFIG" XDG_DATA_HOME="$DATA" "$BIN/cdd" scan

# A few real Jumps on top of the Scan so the ordering shows both sources.
hist="$DATA/cdd/history"
printf '%s\tjump\t%s\n' "$(ago 0)" tools/cdd            >> "$hist"
printf '%s\tjump\t%s\n' "$(ago 1)" domain/kryft.dev     >> "$hist"
printf '%s\tjump\t%s\n' "$(ago 2)" ops/internal         >> "$hist"

cat <<MSG

Demo environment ready in $DEMO_DIR

Try it in a shell:
  set -gx XDG_CONFIG_HOME $CONFIG; set -gx XDG_DATA_HOME $DATA   # fish
  export XDG_CONFIG_HOME=$CONFIG XDG_DATA_HOME=$DATA               # bash/zsh
  $BIN/cdd

Record it:
  vhs demo/demo.tape
MSG
