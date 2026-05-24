package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/zcag/odak/internal/model"
	"github.com/zcag/odak/internal/parser"
)

func fixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/todos.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// parse → write with no mutations → identical bytes.
func TestRoundTripNoChange(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)
	got := parser.Write(f)
	if got != original {
		t.Errorf("round-trip diff:\n%s", lineDiffReport(original, got))
	}
}

// toggling done only changes the checkbox character, nothing else.
func TestToggleDoneMinimalDiff(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	item := f.Items[0]
	item.Done = true
	item.MarkToggled()

	got := parser.Write(f)
	diffs := changedLines(original, got)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 changed line, got %d:\n%s", len(diffs), strings.Join(diffs, "\n"))
	}
	if !strings.Contains(diffs[0], "- [x]") {
		t.Errorf("changed line missing [x]: %q", diffs[0])
	}
}

// items that were not touched stay byte-identical after a sibling toggle.
func TestUntouchedItemsUnchanged(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	last := f.Items[len(f.Items)-1]
	last.Done = true
	last.MarkToggled()

	got := parser.Write(f)
	if len(changedLines(original, got)) != 1 {
		t.Errorf("expected exactly 1 changed line, got %d", len(changedLines(original, got)))
	}
}

// adding an item inserts it in the right section without disturbing other lines.
func TestAddItem(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	newItem := parser.ParseItem("new task", "Next", false, 0)
	f.Items = append(f.Items, newItem)

	got := parser.Write(f)

	if !strings.Contains(got, "- [ ] new task") {
		t.Fatal("added item not in output")
	}

	lines := strings.Split(got, "\n")
	nextIdx, newIdx, boundaryIdx := -1, -1, len(lines)
	for i, l := range lines {
		switch {
		case l == "## Next":
			nextIdx = i
		case strings.Contains(l, "new task"):
			newIdx = i
		}
	}
	// find section boundary after Next
	for i := nextIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") || strings.HasPrefix(lines[i], "## ") {
			boundaryIdx = i
			break
		}
	}
	if newIdx <= nextIdx || newIdx >= boundaryIdx {
		t.Errorf("item not inside Next section (next=%d item=%d boundary=%d)", nextIdx, newIdx, boundaryIdx)
	}
}

// deleting an item removes exactly that line; all others survive unchanged.
func TestDeleteItem(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	target := f.Items[0]
	targetText := target.Text
	f.Items = f.Items[1:]

	got := parser.Write(f)

	if strings.Contains(got, targetText) {
		t.Errorf("deleted item %q still present", targetText)
	}
	for _, item := range f.Items {
		if !strings.Contains(got, item.Text) {
			t.Errorf("surviving item %q missing", item.Text)
		}
	}
}

// moving an item places it in the target section; non-moved items unchanged.
func TestMoveItem(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	var target *model.Item
	for _, item := range f.Items {
		if item.Section == "Next" {
			target = item
			break
		}
	}
	if target == nil {
		t.Skip("no Next items in fixture")
	}
	targetText := target.Text
	target.Section = "Today"
	target.MarkDirty()

	got := parser.Write(f)

	lines := strings.Split(got, "\n")
	todayIdx, nextIdx, itemIdx := -1, -1, -1
	for i, l := range lines {
		switch {
		case l == "## Today":
			todayIdx = i
		case l == "## Next":
			nextIdx = i
		case strings.Contains(l, targetText):
			itemIdx = i
		}
	}
	if itemIdx < 0 {
		t.Fatal("moved item not found in output")
	}
	if itemIdx <= todayIdx || itemIdx >= nextIdx {
		t.Errorf("moved item not in Today section (today=%d item=%d next=%d)", todayIdx, itemIdx, nextIdx)
	}
}

// section headers must appear in the original order.
func TestSectionOrderPreserved(t *testing.T) {
	original := fixture(t)
	f := parser.Parse(original)

	// add an item to a new section
	newItem := parser.ParseItem("something", "Custom", false, 0)
	f.Items = append(f.Items, newItem)
	f.SectionOrder = append(f.SectionOrder, "Custom")

	got := parser.Write(f)

	order := []string{"## Focus", "## Today", "## Next", "## Backlog", "## Someday", "## Recurring", "## Inbox"}
	prev := -1
	for _, header := range order {
		idx := strings.Index(got, header)
		if idx < prev {
			t.Errorf("section %q appeared out of order", header)
		}
		if idx > prev {
			prev = idx
		}
	}
}

// --- helpers ---

func changedLines(a, b string) []string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	n := len(al)
	if len(bl) > n {
		n = len(bl)
	}
	var out []string
	for i := 0; i < n; i++ {
		la, lb := lineAt(al, i), lineAt(bl, i)
		if la != lb {
			out = append(out, lb)
		}
	}
	return out
}

func lineDiffReport(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	n := len(al)
	if len(bl) > n {
		n = len(bl)
	}
	var sb strings.Builder
	for i := 0; i < n; i++ {
		la, lb := lineAt(al, i), lineAt(bl, i)
		if la != lb {
			sb.WriteString("< " + la + "\n")
			sb.WriteString("> " + lb + "\n")
		}
	}
	return sb.String()
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}
