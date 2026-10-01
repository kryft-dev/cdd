package action

import (
	"errors"
	"slices"
	"testing"
)

func str(s string) *string { return &s }
func flag(b bool) *bool    { return &b }

// withBuiltins registers fake built-ins for one test.
func withBuiltins(t *testing.T, as ...Action) {
	t.Helper()
	old := builtins
	builtins = as
	t.Cleanup(func() { builtins = old })
}

func names(as []Action) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Name
	}
	return out
}

func TestMerge_UserActionsFollowBuiltinsSortedByName(t *testing.T) {
	withBuiltins(t, Action{Name: "files", Key: "ctrl+o", Run: "xdg-open {path}", Detach: true})

	got, err := Merge(map[string]Override{
		"zed":  {Run: str("zed {path}")},
		"code": {Key: str("ctrl+v"), Run: str("code {path}"), Detach: flag(true)},
	}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if want := []string{"files", "code", "zed"}; !slices.Equal(names(got), want) {
		t.Fatalf("names = %v, want %v", names(got), want)
	}
	code := got[1]
	if code.Key != "ctrl+v" || code.Run != "code {path}" || !code.Detach || code.Jump {
		t.Errorf("code = %+v", code)
	}
}

func TestMerge_OverridesABuiltinFieldByField(t *testing.T) {
	withBuiltins(t, Action{Name: "files", Key: "ctrl+o", Run: "xdg-open {path}", Detach: true})

	got, err := Merge(map[string]Override{"files": {Key: str("ctrl+f")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	want := Action{Name: "files", Key: "ctrl+f", Run: "xdg-open {path}", Detach: true}
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %+v, want [%+v]", got, want)
	}
}

func TestMerge_EmptyKeyUnbindsABuiltin(t *testing.T) {
	withBuiltins(t, Action{Name: "files", Key: "ctrl+o", Run: "xdg-open {path}"})

	got, err := Merge(map[string]Override{"files": {Key: str("")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got[0].Key != "" {
		t.Errorf("Key = %q, want it unbound", got[0].Key)
	}
}

func TestMerge_Errors(t *testing.T) {
	tests := []struct {
		name      string
		user      map[string]Override
		vim       bool
		wantName  string
		wantField string
	}{
		{"missing run", map[string]Override{"a": {Key: str("ctrl+a")}}, false, "a", "run"},
		{"unknown key", map[string]Override{"a": {Key: str("ctrl+"), Run: str("x")}}, false, "a", "key"},
		{"esc", map[string]Override{"a": {Key: str("esc"), Run: str("x")}}, true, "a", "key"},
		{"ctrl+c", map[string]Override{"a": {Key: str("ctrl+c"), Run: str("x")}}, false, "a", "key"},
		{"printable without vim", map[string]Override{"a": {Key: str("?"), Run: str("x")}}, false, "a", "key"},
		{
			"two Actions on a key",
			map[string]Override{"a": {Key: str("ctrl+x"), Run: str("x")}, "b": {Key: str("ctrl+x"), Run: str("y")}},
			false, "b", "key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withBuiltins(t)
			_, err := Merge(tt.user, tt.vim)
			var ae *Error
			if !errors.As(err, &ae) {
				t.Fatalf("err = %v, want *Error", err)
			}
			if ae.Name != tt.wantName || ae.Field != tt.wantField {
				t.Errorf("Error = %s/%s, want %s/%s", ae.Name, ae.Field, tt.wantName, tt.wantField)
			}
		})
	}
}

func TestMerge_UserKeyDisplacesABuiltinHoldingIt(t *testing.T) {
	withBuiltins(t, Action{Name: "files", Key: "ctrl+o", Run: "xdg-open {path}"})

	got, err := Merge(map[string]Override{"code": {Key: str("ctrl+o"), Run: str("code")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	keys := map[string]string{}
	for _, a := range got {
		keys[a.Name] = a.Key
	}
	if keys["code"] != "ctrl+o" || keys["files"] != "" {
		t.Errorf("keys = %v, want code on ctrl+o and files unbound", keys)
	}
}

func TestMerge_UserRebindingABuiltinDisplacesAnotherBuiltin(t *testing.T) {
	withBuiltins(t,
		Action{Name: "jump", Key: "enter", Jump: true},
		Action{Name: "remote", Key: "ctrl+r", Run: "open-remote {path}"},
	)

	got, err := Merge(map[string]Override{"remote": {Key: str("enter")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	keys := map[string]string{}
	for _, a := range got {
		keys[a.Name] = a.Key
	}
	if keys["remote"] != "enter" || keys["jump"] != "" {
		t.Errorf("keys = %v, want remote on enter and jump unbound", keys)
	}
}

func TestMerge_UserRebindingAnEarlierBuiltinDisplacesALaterOne(t *testing.T) {
	withBuiltins(t,
		Action{Name: "a", Key: "ctrl+a", Run: "x"},
		Action{Name: "b", Key: "ctrl+b", Run: "y"},
	)

	got, err := Merge(map[string]Override{"a": {Key: str("ctrl+b")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got[0].Key != "ctrl+b" || got[1].Key != "" {
		t.Errorf("keys = %q, %q, want a on ctrl+b and b unbound", got[0].Key, got[1].Key)
	}
}

func TestMerge_TwoUserActionsOnEnterIsAnError(t *testing.T) {
	_, err := Merge(map[string]Override{
		"code": {Key: str("enter"), Run: str("code {path}")},
		"edit": {Key: str("enter"), Run: str("vi {path}")},
	}, false)
	var ae *Error
	if !errors.As(err, &ae) || ae.Name != "edit" || ae.Field != "key" {
		t.Fatalf("err = %v, want an *Error for edit's key", err)
	}
}

func TestMerge_PrintableKeyAllowedWithVim(t *testing.T) {
	withBuiltins(t)
	got, err := Merge(map[string]Override{"a": {Key: str("?"), Run: str("x")}}, true)
	if err != nil || got[0].Key != "?" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestMerge_JumpNeedsNoRun(t *testing.T) {
	withBuiltins(t, Action{Name: "jump", Key: "enter", Jump: true})
	if _, err := Merge(nil, false); err != nil {
		t.Fatalf("Merge: %v", err)
	}
}

func TestBuiltins_JumpIsBoundToEnterAndNeedsNoCommand(t *testing.T) {
	var got *Action
	for _, a := range Builtins() {
		if a.Name == "jump" {
			got = &a
		}
	}
	if got == nil || got.Key != "enter" || got.Run != "" || !got.Jump || got.Detach {
		t.Fatalf("jump built-in = %+v, want a Jump bound to enter with no command", got)
	}
}

func TestMerge_JumpRebindsAndEnterGoesToAnotherAction(t *testing.T) {
	detach := true
	got, err := Merge(map[string]Override{
		"jump": {Key: str("alt+enter")},
		"code": {Key: str("enter"), Run: str("code {path}"), Detach: &detach},
	}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	keys := map[string]string{}
	for _, a := range got {
		keys[a.Name] = a.Key
	}
	if keys["jump"] != "alt+enter" || keys["code"] != "enter" {
		t.Errorf("keys = %v, want jump on alt+enter and code on enter", keys)
	}
}

func TestMerge_EnterGoesToAnotherActionAndJumpIsLeftUnbound(t *testing.T) {
	got, err := Merge(map[string]Override{"code": {Key: str("enter"), Run: str("code {path}")}}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	keys := map[string]string{}
	for _, a := range got {
		keys[a.Name] = a.Key
	}
	if keys["code"] != "enter" || keys["jump"] != "" {
		t.Errorf("keys = %v, want code on enter and jump unbound", keys)
	}
}

func TestMerge_JumpMayBeLeftUnbound(t *testing.T) {
	got, err := Merge(map[string]Override{"jump": {Key: str("")}}, false)
	if err != nil || got[0].Name != "jump" || got[0].Key != "" || !got[0].Jump {
		t.Fatalf("got %+v, %v, want an unbound jump", got, err)
	}
}
