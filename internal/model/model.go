package model

// Section is a free-form string matching whatever ## headings exist in the file.
type Section = string

type Item struct {
	ID       string   `json:"id"`
	Section  Section  `json:"section"`
	Done     bool     `json:"done"`
	Text     string   `json:"text"`
	Tags     []string `json:"tags,omitempty"`
	Urgent   bool     `json:"urgent,omitempty"`
	Deadline string   `json:"deadline,omitempty"`
	Trigger  string   `json:"trigger,omitempty"`
	FileRef  string   `json:"file_ref,omitempty"`
	ParentID string   `json:"parent_id,omitempty"`
	Children []*Item  `json:"children,omitempty"`
	Depth       int    `json:"depth,omitempty"`
	rawLine     string
	dirty       bool // full re-render needed
	toggledOnly bool // only Done flipped; update checkbox in-place for minimal diff
}

func (i *Item) SetDepth(d int) { i.Depth = d }
func (i *Item) RawLine() string      { return i.rawLine }
func (i *Item) SetRawLine(s string)  { i.rawLine = s }
func (i *Item) MarkDirty()           { i.dirty = true; i.toggledOnly = false }
func (i *Item) MarkToggled()         { if !i.dirty { i.toggledOnly = true } }
func (i *Item) IsDirty() bool        { return i.dirty }
func (i *Item) IsToggledOnly() bool  { return i.toggledOnly }
