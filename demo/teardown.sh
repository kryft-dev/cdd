#!/usr/bin/env bash
# Deletes what demo/setup.sh created, and nothing else: ~/Code, ~/Work,
# and ~/.cache/cdd-demo. It does nothing
# unless setup.sh's marker file is there, so it cannot remove a real ~/Code
# or ~/Work.
set -euo pipefail

DEMO_DIR="$HOME/.cache/cdd-demo"
if [ ! -e "$DEMO_DIR/.created-by-setup" ]; then
  echo "teardown.sh: $DEMO_DIR/.created-by-setup is missing; not deleting anything" >&2
  exit 1
fi
rm -rf -- "$HOME/Code" "$HOME/Work" "$DEMO_DIR"
echo "Removed ~/Code, ~/Work and ~/.cache/cdd-demo"
