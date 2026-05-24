package parser

import (
	"fmt"
	"strings"

	"github.com/zcag/odak/internal/model"
)

// Reorder reorders items within a section to match the given id order.
// ids must be exactly the IDs of all file-tracked items in the section.
func (f *File) Reorder(section string, ids []string) error {
	// Collect (lineIndex, id, rawText) for items in this section that have a raw line.
	type entry struct {
		line int
		id   string
		text string
	}
	var entries []entry
	for _, item := range f.Items {
		if item.Section != section {
			continue
		}
		li, ok := f.itemLine[item.ID]
		if !ok {
			continue
		}
		entries = append(entries, entry{li, item.ID, f.rawLines[li]})
	}
	// Sort by ascending line so entries[i].line is the i-th slot in file order.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].line < entries[j-1].line; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	if len(entries) != len(ids) {
		return fmt.Errorf("reorder: expected %d ids for section %q, got %d", len(entries), section, len(ids))
	}
	// Build id → raw text map.
	idText := make(map[string]string, len(entries))
	for _, e := range entries {
		idText[e.id] = e.text
	}
	for _, id := range ids {
		if _, ok := idText[id]; !ok {
			return fmt.Errorf("reorder: id %s not found in section %q", id, section)
		}
	}
	// Assign each slot the content of the corresponding new id.
	for i, id := range ids {
		targetLine := entries[i].line
		f.rawLines[targetLine] = idText[id]
		f.itemLine[id] = targetLine
	}
	return nil
}

// Write produces the file content with minimal diffs:
//   - unchanged items   → original line verbatim (zero diff)
//   - toggled-done only → only the [ ]/[x] character replaced in the raw line
//   - dirty items       → full line re-rendered in normalized format
//   - deleted items     → line removed
//   - new/moved items   → inserted after last item in target section
//   - all other lines   → verbatim (separators, blank lines, preamble, comments)
func Write(f *File) string {
	lines := make([]string, len(f.rawLines))
	copy(lines, f.rawLines)

	deleted := make(map[int]bool)
	insertAfter := make(map[int][]string) // rawLines index → new lines to emit after it

	currentIDs := make(map[string]bool)
	for _, item := range f.Items {
		currentIDs[item.ID] = true
	}

	for id, li := range f.itemLine {
		if !currentIDs[id] {
			deleted[li] = true
		}
	}

	for _, item := range f.Items {
		li, existed := f.itemLine[item.ID]

		if !existed {
			pt := f.sectionInsertPoint(item.Section)
			insertAfter[pt] = append(insertAfter[pt], renderItem(item))
			continue
		}

		if f.lineSection[li] != item.Section {
			deleted[li] = true
			pt := f.sectionInsertPoint(item.Section)
			insertAfter[pt] = append(insertAfter[pt], renderItem(item))
			continue
		}

		switch {
		case item.IsDirty():
			lines[li] = renderItem(item)
		case item.IsToggledOnly():
			lines[li] = toggleCheckbox(f.rawLines[li], item.Done)
		}
	}

	var sb strings.Builder
	for i, line := range lines {
		if !deleted[i] {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		for _, ins := range insertAfter[i] {
			sb.WriteString(ins)
			sb.WriteString("\n")
		}
	}

	out := sb.String()
	if !f.trailingNL {
		out = strings.TrimSuffix(out, "\n")
	}
	return out
}

func renderItem(item *model.Item) string {
	var sb strings.Builder
	sb.WriteString(strings.Repeat("  ", item.Depth))
	if item.Done {
		sb.WriteString("- [x] ")
	} else {
		sb.WriteString("- [ ] ")
	}
	for _, tag := range item.Tags {
		sb.WriteString("[t:")
		sb.WriteString(tag)
		sb.WriteString("] ")
	}
	if item.Urgent {
		sb.WriteString("[!] ")
	}
	if item.Deadline != "" {
		sb.WriteString("[d:")
		sb.WriteString(item.Deadline)
		sb.WriteString("] ")
	}
	if item.Trigger != "" {
		sb.WriteString("[w:")
		sb.WriteString(item.Trigger)
		sb.WriteString("] ")
	}
	sb.WriteString(item.Text)
	if item.FileRef != "" {
		sb.WriteString(" [→ ")
		sb.WriteString(item.FileRef)
		sb.WriteString("]")
	}
	return sb.String()
}

func toggleCheckbox(raw string, nowDone bool) string {
	if nowDone {
		return strings.Replace(raw, "- [ ]", "- [x]", 1)
	}
	return strings.Replace(raw, "- [x]", "- [ ]", 1)
}
