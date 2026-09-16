package history

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
)

// formatLine renders v as one tab-separated History line, RFC 3339 UTC
// second precision, ending in a newline.
func formatLine(v Visit) string {
	return v.At.UTC().Truncate(time.Second).Format(time.RFC3339) + "\t" + string(v.Source) + "\t" + v.Project + "\n"
}

// parseLine parses one History line into a Visit. It reports ok=false for
// any malformed line: wrong field count, an unparseable timestamp, an
// unknown source, or an empty Project.
func parseLine(line string) (v Visit, ok bool) {
	fields := strings.Split(line, "\t")
	if len(fields) != 3 {
		return Visit{}, false
	}

	at, err := time.Parse(time.RFC3339, fields[0])
	if err != nil {
		return Visit{}, false
	}

	var src Source
	switch fields[1] {
	case string(SourceJump):
		src = SourceJump
	case string(SourceScan):
		src = SourceScan
	default:
		return Visit{}, false
	}

	project := fields[2]
	if project == "" {
		return Visit{}, false
	}

	return Visit{At: at.UTC(), Source: src, Project: project}, true
}

// readVisits reads and parses every line of the History file at path,
// skipping malformed lines silently. A missing file yields (nil, nil).
func readVisits(path string) ([]Visit, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return parseVisits(f)
}

// parseVisits reads and parses every line from r, skipping malformed lines
// silently.
func parseVisits(r io.Reader) ([]Visit, error) {
	var visits []Visit
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if v, ok := parseLine(line); ok {
			visits = append(visits, v)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return visits, nil
}
