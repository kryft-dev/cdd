package cli

import (
	"runtime/debug"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	vcs := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "b72f0117d2c4a9e1f0c3"},
			{Key: "vcs.time", Value: "2026-09-17T07:20:28Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}
	dirty := &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "b72f011"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	tests := []struct {
		name                  string
		version, commit, date string
		info                  *debug.BuildInfo
		want                  string
	}{
		{name: "ldflags from GoReleaser", version: "0.1.1", commit: "b72f011", date: "2026-09-17", info: vcs, want: "cdd 0.1.1 (b72f011 2026-09-17)"},
		{name: "go install of a tagged module", version: "dev", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}}, want: "cdd 0.1.1 (go install)"},
		{name: "build from a clean checkout", version: "dev", info: vcs, want: "cdd dev (b72f011 2026-09-17)"},
		{name: "build from a dirty checkout", version: "dev", info: dirty, want: "cdd dev (b72f011+dirty)"},
		{name: "no build information", version: "dev", info: nil, want: "cdd dev (dev)"},
		{name: "devel without vcs", version: "dev", info: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, want: "cdd dev (dev)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.version, tt.commit, tt.date, tt.info)
			if got != tt.want {
				t.Errorf("formatVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
