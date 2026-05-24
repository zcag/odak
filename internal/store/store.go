package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zcag/odak/internal/model"
	"github.com/zcag/odak/internal/parser"
)

type Store struct {
	path      string
	backupDir string
	writing   atomic.Bool
}

func New(path, backupDir string) *Store {
	return &Store{path: path, backupDir: backupDir}
}

func (s *Store) Read() (*parser.File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	return parser.Parse(string(data)), nil
}

func (s *Store) Write(f *parser.File) error {
	if err := s.backup(); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	lf, err := s.lock()
	if err != nil {
		return err
	}
	defer s.unlock(lf)

	s.writing.Store(true)
	defer s.writing.Store(false)
	return os.WriteFile(s.path, []byte(parser.Write(f)), 0644)
}

func (s *Store) WriteRaw(content string) error {
	if err := s.backup(); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	lf, err := s.lock()
	if err != nil {
		return err
	}
	defer s.unlock(lf)

	s.writing.Store(true)
	defer s.writing.Store(false)
	return os.WriteFile(s.path, []byte(content), 0644)
}

// WatchFile calls onChange whenever the file is modified externally (not by this server).
// Runs until the process exits.
func (s *Store) WatchFile(onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(s.path); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer w.Close()
		var debounce *time.Timer
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) {
					continue
				}
				if s.writing.Load() {
					continue
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(80*time.Millisecond, onChange)
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return nil
}

func (s *Store) ReadRaw() (string, error) {
	data, err := os.ReadFile(s.path)
	return string(data), err
}

func (s *Store) backup() error {
	if s.backupDir == "" {
		return nil
	}
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	ts := time.Now().Format("2006-01-02T15-04-05")
	base := filepath.Base(s.path)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + "_" + ts + ext
	return os.WriteFile(filepath.Join(s.backupDir, name), data, 0644)
}

func (s *Store) lock() (*os.File, error) {
	lf, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		lf.Close()
		return nil, err
	}
	return lf, nil
}

func (s *Store) unlock(lf *os.File) {
	syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
	lf.Close()
}

// Mutate reads, applies fn, then writes.
func (s *Store) Mutate(fn func(*parser.File) error) error {
	f, err := s.Read()
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	return s.Write(f)
}

// AddItem adds an item and returns it.
func (s *Store) AddItem(item *model.Item) (*model.Item, error) {
	var added *model.Item
	err := s.Mutate(func(f *parser.File) error {
		f.Items = append(f.Items, item)
		added = item
		return nil
	})
	return added, err
}

// UpdateItem finds by ID and applies patch fields.
func (s *Store) UpdateItem(id string, patch *model.Item) (*model.Item, error) {
	var updated *model.Item
	err := s.Mutate(func(f *parser.File) error {
		item := f.ByID(id)
		if item == nil {
			return fmt.Errorf("not found: %s", id)
		}
		if patch.Text != "" {
			item.Text = patch.Text
		}
		if patch.Section != "" {
			item.Section = patch.Section
		}
		if patch.Tags != nil {
			item.Tags = patch.Tags
		}
		if patch.Deadline != "" {
			item.Deadline = patch.Deadline
		}
		if patch.Trigger != "" {
			item.Trigger = patch.Trigger
		}
		item.Urgent = patch.Urgent
		item.MarkDirty()
		updated = item
		return nil
	})
	return updated, err
}

// ToggleDone flips the done state.
func (s *Store) ToggleDone(id string) (*model.Item, error) {
	var result *model.Item
	err := s.Mutate(func(f *parser.File) error {
		item := f.ByID(id)
		if item == nil {
			return fmt.Errorf("not found: %s", id)
		}
		item.Done = !item.Done
		item.MarkToggled()
		result = item
		return nil
	})
	return result, err
}

// DeleteItem removes an item and its children.
func (s *Store) DeleteItem(id string) error {
	return s.Mutate(func(f *parser.File) error {
		var keep []*model.Item
		for _, item := range f.Items {
			if item.ID != id && item.ParentID != id {
				keep = append(keep, item)
			}
		}
		if len(keep) == len(f.Items) {
			return fmt.Errorf("not found: %s", id)
		}
		f.Items = keep
		return nil
	})
}

// ReorderItems reorders items within a section to match the given id slice.
func (s *Store) ReorderItems(section string, ids []string) error {
	return s.Mutate(func(f *parser.File) error {
		return f.Reorder(section, ids)
	})
}

// MoveItem changes the section of an item.
func (s *Store) MoveItem(id string, section model.Section) (*model.Item, error) {
	var result *model.Item
	err := s.Mutate(func(f *parser.File) error {
		item := f.ByID(id)
		if item == nil {
			return fmt.Errorf("not found: %s", id)
		}
		item.Section = section
		item.MarkDirty()
		result = item
		return nil
	})
	return result, err
}
