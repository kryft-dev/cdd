package action_test

import (
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

func TestBuiltins_CopyIsBoundToCtrlYWithNoCommand(t *testing.T) {
	got := builtin(t, "copy")
	want := action.Action{Name: "copy", Key: "ctrl+y", Copy: true}
	if got != want {
		t.Errorf("copy = %+v, want %+v", got, want)
	}
}

func TestMerge_CopyKeepsCopyUntilTheUserSetsRun(t *testing.T) {
	merged := func(o action.Override) action.Action {
		t.Helper()
		got, err := action.Merge(map[string]action.Override{"copy": o}, nil, false)
		if err != nil {
			t.Fatalf("Merge: %v", err)
		}
		for _, a := range got {
			if a.Name == "copy" {
				return a
			}
		}
		t.Fatal("no copy")
		return action.Action{}
	}

	if a := merged(action.Override{Key: ptr("ctrl+k")}); !a.Copy || a.Key != "ctrl+k" {
		t.Errorf("rebound = %+v, want still Copy on ctrl+k", a)
	}
	if a := merged(action.Override{Key: ptr("")}); !a.Copy || a.Key != "" {
		t.Errorf("unbound = %+v, want still Copy, no key", a)
	}
	if a := merged(action.Override{Run: ptr("wl-copy {path}"), Detach: ptr(true)}); a.Copy || a.Run != "wl-copy {path}" || !a.Detach {
		t.Errorf("run override = %+v, want a plain command, not Copy", a)
	}
}
