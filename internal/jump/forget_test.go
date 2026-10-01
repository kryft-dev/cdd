package jump_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
)

func TestResolve_PickerForgetWritesTheStore(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd", "tools/other")
	path := filepath.Join(root, "tools", "cdd")
	store := newStore(t)

	pick := func(_ []picker.Row, _ picker.StatusFunc, o picker.Options) (picker.Choice, bool, error) {
		if o.Forget == nil {
			t.Fatal("Options.Forget = nil, want the Store's")
		}
		if err := o.Forget(path); err != nil {
			t.Fatalf("Forget: %v", err)
		}
		return picker.Choice{}, false, nil
	}
	_, err := jump.Resolve(context.Background(), cfg, hist, store, pick, &fakeRunner{})

	if err != jump.ErrCancelled {
		t.Fatalf("Resolve error = %v, want ErrCancelled", err)
	}
	marks, err := store.Marks()
	if err != nil {
		t.Fatalf("Marks: %v", err)
	}
	if !marks.IsForgotten(path) {
		t.Errorf("%s is not Forgotten after the Picker forgot it", path)
	}
	if marks.IsForgotten(filepath.Join(root, "tools", "other")) {
		t.Error("another Project is Forgotten too")
	}
}
