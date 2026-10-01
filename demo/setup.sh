#!/usr/bin/env bash
# Builds the demo environment that demo/demo.tape records against.
#
#   demo/setup.sh && vhs demo/demo.tape; demo/teardown.sh
#
# It creates, in your REAL home directory:
#   ~/Code  ~/Work     stand-in projects: empty `git init` dirs with one
#                      dated commit and an origin remote, plus ~/Work/notes
#                      (not a repository, for `cdd add`)
#   ~/.cache/cdd-demo  cdd's config, History and projects file (XDG_*_HOME
#                      point here, so your real ones are never touched), and
#                      bin/ holding the cdd built from this checkout and
#                      stubs for xdg-open, open, lazygit and the clipboard
#
# It refuses to run if any of those three directories already exists, and
# demo/teardown.sh deletes exactly those three and nothing else.
set -euo pipefail

CODE="$HOME/Code"
WORK="$HOME/Work"
DEMO_DIR="$HOME/.cache/cdd-demo"
for d in "$CODE" "$WORK" "$DEMO_DIR"; do
  if [ -e "$d" ] || [ -L "$d" ]; then
    echo "setup.sh: $d already exists; refusing to run" >&2
    exit 1
  fi
done

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="$DEMO_DIR/config"
DATA="$DEMO_DIR/data"   # created by cdd on first write
STATE="$DEMO_DIR/state"
BIN="$DEMO_DIR/bin"
mkdir -p "$DEMO_DIR" "$DATA" "$CODE" "$WORK" "$CONFIG/cdd" "$STATE" "$BIN"
touch "$DEMO_DIR/.created-by-setup"   # teardown deletes nothing without it

# --- Projects -----------------------------------------------------------
git_demo() { git -c user.name=demo -c user.email=demo@example.com -c commit.gpgsign=false "$@"; }

ago() { # ago <days> -> ISO timestamp, GNU or BSD date
  date -u -d "$1 days ago" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null \
    || date -u -v-"$1"d +%Y-%m-%dT%H:%M:%SZ
}

repo() { # repo <dir> <days-ago> <remote>
  local dir="$1" when; when="$(ago "$2")"
  mkdir -p "$dir"
  git_demo -C "$dir" init -q -b main
  echo "# ${dir##*/}" > "$dir/README.md"
  git_demo -C "$dir" add -A
  GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" git_demo -C "$dir" commit -q -m "initial commit"
  git_demo -C "$dir" remote add origin "$3"
}

repo "$CODE/next.js"        1  git@github.com:vercel/next.js.git
repo "$CODE/go"             2  git@github.com:golang/go.git
repo "$CODE/react"          4  git@github.com:facebook/react.git
repo "$CODE/rust"           7  git@github.com:rust-lang/rust.git
repo "$CODE/oss/kubernetes" 3  git@github.com:kubernetes/kubernetes.git
repo "$CODE/oss/deno"       9  git@github.com:denoland/deno.git
repo "$WORK/acme/api"       0  git@github.com:acme/api.git
repo "$WORK/acme/web"       5  git@github.com:acme/web.git
repo "$WORK/globex/billing" 6  git@github.com:globex/billing.git
mkdir -p "$WORK/notes"      # not a repository: shown by `cdd add`

# --- Binary and stubs ----------------------------------------------------
# Nothing in the demo opens a browser, a file manager or a real lazygit.
( cd "$REPO_DIR" && go build -o "$BIN/cdd.real" ./cmd/cdd )

# cdd, with $HOME shown as ~ in what it prints, so the recording does not
# show the username. The Picker (`cdd pick`) is passed straight through.
cat > "$BIN/cdd" <<SH
#!/bin/sh
case "\$1" in
  add|forget|config|scan) "$BIN/cdd.real" "\$@" 2>&1 | sed "s|$HOME|~|g" ;;
  *) exec "$BIN/cdd.real" "\$@" ;;
esac
SH

for opener in xdg-open open; do
  cat > "$BIN/$opener" <<SH
#!/bin/sh
echo "$opener \$*" >> "$DEMO_DIR/opened.log"
SH
done
for clip in wl-copy xclip xsel pbcopy; do
  printf '#!/bin/sh\ncat > /dev/null\n' > "$BIN/$clip"
done
cat > "$BIN/lazygit" <<'SH'
#!/bin/sh
printf '\n  lazygit (demo stand-in)\n\n'
printf '  Status    main, up to date with origin/main\n'
printf '  Files     nothing to commit, working tree clean\n'
printf '  Branches  * main\n'
printf '  Commits   initial commit\n\n'
sleep 1.5
SH
chmod +x "$BIN"/*

# --- Config, History and the demo's own Action -----------------------------
cat > "$CONFIG/cdd/config.toml" <<'TOML'
[actions.lazygit]
key = "ctrl+l"
run = "lazygit"
TOML

cat <<MSG
Demo environment ready in $DEMO_DIR, with projects in ~/Code and ~/Work.

Record it (VHS v0.11.0; v0.12.0 silently writes no GIF), then clean up:
  vhs demo/demo.tape
  demo/teardown.sh
MSG
