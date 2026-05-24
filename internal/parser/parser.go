package parser

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/zcag/odak/internal/model"
)

var (
	sectionRe  = regexp.MustCompile(`^## (.+)$`)
	itemRe     = regexp.MustCompile(`^(\s*)- \[([ x])\] (.*)$`)
	tagRe      = regexp.MustCompile(`\[t:([^\]]+)\]`)
	urgentRe   = regexp.MustCompile(`\[!\]`)
	deadlineRe = regexp.MustCompile(`\[d:([^\]]+)\]`)
	triggerRe  = regexp.MustCompile(`\[w:([^\]]+)\]`)
	fileRefRe  = regexp.MustCompile(`\[→ ([^\]]+)\]`)
	allMetaRe  = regexp.MustCompile(`\s*\[(?:t:[^\]]+|!|d:[^\]]+|w:[^\]]+|→ [^\]]+)\]`)
)

// File is a parsed todos.md with full line tracking for round-trip safe writes.
type File struct {
	rawLines      []string
	trailingNL    bool           // original file ended with \n
	SectionOrder  []string       // sections in file order
	Items         []*model.Item
	itemLine      map[string]int // item ID → rawLines index
	lineSection   map[int]string // rawLines index → section name (for move detection)
	sectionHeader map[string]int // section name → header line index
}

func Parse(content string) *File {
	trailingNL := strings.HasSuffix(content, "\n")
	raw := strings.TrimRight(content, "\n")
	lines := strings.Split(raw, "\n")

	f := &File{
		rawLines:      lines,
		trailingNL:    trailingNL,
		itemLine:      make(map[string]int),
		lineSection:   make(map[int]string),
		sectionHeader: make(map[string]int),
	}

	var currentSection string
	inSection := false
	parentStack := make([]*model.Item, 20)
	seenSections := map[string]bool{}

	for i, line := range lines {
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			currentSection = strings.TrimSpace(m[1])
			inSection = true
			parentStack = make([]*model.Item, 20)
			if !seenSections[currentSection] {
				seenSections[currentSection] = true
				f.SectionOrder = append(f.SectionOrder, currentSection)
				f.sectionHeader[currentSection] = i
			}
			continue
		}
		if !inSection {
			continue
		}
		m := itemRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		depth := len(m[1]) / 2
		done := m[2] == "x"
		raw := m[3]

		item := parseContent(raw, currentSection, done, depth)
		item.SetRawLine(line)

		if depth > 0 && depth < len(parentStack) && parentStack[depth-1] != nil {
			item.ParentID = parentStack[depth-1].ID
		}
		if depth < len(parentStack) {
			parentStack[depth] = item
			for j := depth + 1; j < len(parentStack); j++ {
				parentStack[j] = nil
			}
		}
		f.Items = append(f.Items, item)
		f.itemLine[item.ID] = i
		f.lineSection[i] = currentSection
	}

	return f
}

// ParseItem parses a single raw item string — exported for ID derivation without a full file.
func ParseItem(raw string, section model.Section, done bool, depth int) *model.Item {
	return parseContent(raw, section, done, depth)
}

func parseContent(raw string, section model.Section, done bool, depth int) *model.Item {
	item := &model.Item{
		Section: section,
		Done:    done,
	}
	item.SetDepth(depth)

	for _, m := range tagRe.FindAllStringSubmatch(raw, -1) {
		item.Tags = append(item.Tags, m[1])
	}
	if urgentRe.MatchString(raw) {
		item.Urgent = true
	}
	if m := deadlineRe.FindStringSubmatch(raw); m != nil {
		item.Deadline = m[1]
	}
	if m := triggerRe.FindStringSubmatch(raw); m != nil {
		item.Trigger = m[1]
	}
	if m := fileRefRe.FindStringSubmatch(raw); m != nil {
		item.FileRef = m[1]
	}
	item.Text = strings.TrimSpace(allMetaRe.ReplaceAllString(raw, ""))

	h := sha256.Sum256([]byte(raw))
	item.ID = fmt.Sprintf("%x", h[:4])
	return item
}

func (f *File) Flat() []*model.Item { return f.Items }

func (f *File) ByID(id string) *model.Item {
	for _, item := range f.Items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

// sectionInsertPoint returns the rawLines index after which a new item should be inserted
// for the given section. It is the last item line in the section, or the line after the
// section header if the section is empty.
func (f *File) sectionInsertPoint(section string) int {
	last := -1
	for _, item := range f.Items {
		li, ok := f.itemLine[item.ID]
		if !ok {
			continue
		}
		// Use lineSection (original file position), not item.Section (may be mid-mutation).
		if f.lineSection[li] == section && li > last {
			last = li
		}
	}
	if last >= 0 {
		return last
	}
	if hi, ok := f.sectionHeader[section]; ok {
		// after header + blank line
		if hi+1 < len(f.rawLines) {
			return hi + 1
		}
		return hi
	}
	return len(f.rawLines) - 1
}
