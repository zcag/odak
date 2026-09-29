package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zcag/odak/internal/model"
)

func ptr[T any](v T) *T { return &v }

func TestUpdateReturnsUsableID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.md")
	os.WriteFile(path, []byte("## Inbox\n\n- [ ] [t:a] [!] [d:2026-01-01] old\n"), 0644)
	s := New(path, "")

	f, _ := s.Read()
	id := f.Items[0].ID

	// omitted fields (tags, urgent) must survive; "" clears the deadline
	it, err := s.UpdateItem(id, &model.Patch{Text: ptr("new"), Deadline: ptr("")})
	if err != nil {
		t.Fatal(err)
	}
	if !it.Urgent || len(it.Tags) != 1 || it.Deadline != "" {
		t.Fatalf("patch applied wrong: %+v", it)
	}

	f, _ = s.Read()
	if f.Items[0].ID != it.ID {
		t.Fatalf("returned id %s, file has %s", it.ID, f.Items[0].ID)
	}
	moved, err := s.MoveItem(it.ID, "Inbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteItem(moved.ID); err != nil {
		t.Fatal(err)
	}
}
