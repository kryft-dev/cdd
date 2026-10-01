package action_test

import (
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

func TestBuiltins_ForgetIsBoundToCtrlDWithNoCommand(t *testing.T) {
	got := builtin(t, "forget")
	want := action.Action{Name: "forget", Key: "ctrl+d", Internal: action.InternalForget}
	if got != want {
		t.Errorf("forget = %+v, want %+v", got, want)
	}
}

func TestMerge_ForgetKeepsItsJobUntilTheUserSetsRun(t *testing.T) {
	merged := func(o action.Override) action.Action {
		t.Helper()
		got, err := action.Merge(map[string]action.Override{"forget": o}, nil, false)
		if err != nil {
			t.Fatalf("Merge: %v", err)
		}
		for _, a := range got {
			if a.Name == "forget" {
				return a
			}
		}
		t.Fatal("no forget")
		return action.Action{}
	}

	if a := merged(action.Override{Key: ptr("ctrl+k")}); a.Internal != action.InternalForget || a.Key != "ctrl+k" {
		t.Errorf("rebound = %+v, want still forget on ctrl+k", a)
	}
	if a := merged(action.Override{Key: ptr("")}); a.Internal != action.InternalForget || a.Key != "" {
		t.Errorf("unbound = %+v, want still forget, no key", a)
	}
	if a := merged(action.Override{Run: ptr("echo {path}")}); a.Internal != "" || a.Run != "echo {path}" {
		t.Errorf("run override = %+v, want a plain command", a)
	}
}
