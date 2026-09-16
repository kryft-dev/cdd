package git

import (
	"bytes"
	"strconv"
	"strings"
)

// parseStatus classifies the NUL-separated output of
// `git status --porcelain=v2 --branch -unormal -z`. Only the first byte of
// each record matters for the worktree state; branch.* headers carry the
// sync state.
func parseStatus(out []byte) Status {
	st := Status{Kind: Found, State: Clean}

	records := bytes.Split(out, []byte{0})
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) == 0 {
			continue
		}
		line := string(rec)

		switch {
		case strings.HasPrefix(line, "# branch.head "):
			st.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream "):
			st.HasUpstream = true
		case strings.HasPrefix(line, "# branch.ab "):
			if ahead, behind, ok := parseAB(line); ok {
				st.Ahead = ahead
				st.Behind = behind
			}
		case strings.HasPrefix(line, "#"):
			// Other header (branch.oid, stash, or unrecognised): ignore.
		case strings.HasPrefix(line, "2 "):
			// Renamed/copied entry: dirty. Its NUL-separated original path
			// follows as the next record; skip it so it is not misread as
			// its own record.
			st.State = Dirty
			i++
		case line[0] == '1', line[0] == 'u':
			st.State = Dirty
		case line[0] == '?':
			st.Untracked = true
		default:
			// '!' (ignored, only with --ignored) or anything else: ignore.
		}
	}

	return st
}

// parseAB parses a "# branch.ab +A -B" header. ok is false for
// "# branch.ab +? -?", which means no sync info (ahead/behind not computed).
func parseAB(line string) (ahead, behind int, ok bool) {
	fields := strings.Fields(line)
	if len(fields) != 4 {
		return 0, 0, false
	}

	a := strings.TrimPrefix(fields[2], "+")
	b := strings.TrimPrefix(fields[3], "-")
	if a == "?" || b == "?" {
		return 0, 0, false
	}

	ai, errA := strconv.Atoi(a)
	bi, errB := strconv.Atoi(b)
	if errA != nil || errB != nil {
		return 0, 0, false
	}

	return ai, bi, true
}
