package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/zcag/odak/config"
	"github.com/zcag/odak/internal/client"
	"github.com/zcag/odak/internal/model"
)

func newClient() *client.Client {
	cfg := config.LoadClient()
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "odak: no token configured (set ODAK_TOKEN or ~/.config/odak/client)")
		os.Exit(1)
	}
	return client.New(cfg.Endpoint, cfg.Token)
}

// isFilterToken reports whether arg is a tag filter like t:work or t:-work.
func isFilterToken(s string) bool { return strings.HasPrefix(s, "t:") }

func runList(args []string) {
	section := ""
	var inc, exc []string
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "t:-"):
			if t := a[3:]; t != "" {
				exc = append(exc, t)
			}
		case strings.HasPrefix(a, "t:"):
			if t := a[2:]; t != "" {
				inc = append(inc, t)
			}
		default:
			if section == "" {
				section = a
			}
		}
	}
	items, err := newClient().List(section, "", "")
	die(err)
	printItems(filterByTags(items, inc, exc))
}

// filterByTags keeps items with any included tag (OR) and drops any with an
// excluded tag — mirrors the web UI's include/exclude semantics.
func filterByTags(items []*model.Item, inc, exc []string) []*model.Item {
	if len(inc) == 0 && len(exc) == 0 {
		return items
	}
	var out []*model.Item
	for _, it := range items {
		if len(inc) > 0 && !hasAny(it.Tags, inc) {
			continue
		}
		if hasAny(it.Tags, exc) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func hasAny(tags, set []string) bool {
	for _, t := range tags {
		for _, s := range set {
			if t == s {
				return true
			}
		}
	}
	return false
}

func runAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: odak add <text> [--section S] [--tag T] [--urgent] [--deadline D]")
		os.Exit(1)
	}
	// simple inline flag parse: collect --key value pairs from args
	item := &model.Item{Section: "Inbox"}
	var textParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--section", "-s":
			i++
			if i < len(args) {
				item.Section = model.Section(args[i])
			}
		case "--tag", "-t":
			i++
			if i < len(args) {
				item.Tags = append(item.Tags, args[i])
			}
		case "--urgent", "-u":
			item.Urgent = true
		case "--deadline", "-d":
			i++
			if i < len(args) {
				item.Deadline = args[i]
			}
		case "--parent", "-p":
			i++
			if i < len(args) {
				item.ParentID = args[i]
			}
		default:
			textParts = append(textParts, args[i])
		}
	}
	item.Text = strings.Join(textParts, " ")
	if item.Text == "" {
		fmt.Fprintln(os.Stderr, "odak: text is required")
		os.Exit(1)
	}
	created, err := newClient().Create(item)
	die(err)
	fmt.Printf("created %s\n", created.ID)
}

func runDone(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: odak done <id>")
		os.Exit(1)
	}
	item, err := newClient().ToggleDone(args[0])
	die(err)
	state := "done"
	if !item.Done {
		state = "undone"
	}
	fmt.Printf("%s marked %s\n", item.ID, state)
}

func runRm(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: odak rm <id>")
		os.Exit(1)
	}
	die(newClient().Delete(args[0]))
	fmt.Printf("deleted %s\n", args[0])
}

func runMove(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: odak move <id> <section>")
		os.Exit(1)
	}
	item, err := newClient().Move(args[0], model.Section(args[1]))
	die(err)
	fmt.Printf("%s moved to %s\n", item.ID, item.Section)
}

func runShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: odak show <id>")
		os.Exit(1)
	}
	item, err := newClient().Get(args[0])
	die(err)
	fmt.Printf("id:       %s\n", item.ID)
	fmt.Printf("section:  %s\n", item.Section)
	fmt.Printf("done:     %v\n", item.Done)
	fmt.Printf("text:     %s\n", item.Text)
	if len(item.Tags) > 0 {
		fmt.Printf("tags:     %s\n", strings.Join(item.Tags, ", "))
	}
	if item.Urgent {
		fmt.Printf("urgent:   true\n")
	}
	if item.Deadline != "" {
		fmt.Printf("deadline: %s\n", item.Deadline)
	}
	if item.ParentID != "" {
		fmt.Printf("parent:   %s\n", item.ParentID)
	}
}

func printItems(items []*model.Item) {
	if len(items) == 0 {
		fmt.Println("(empty)")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, item := range items {
		done := "[ ]"
		if item.Done {
			done = "[x]"
		}
		prefix := strings.Repeat("  ", item.Depth)
		tags := ""
		if len(item.Tags) > 0 {
			tags = " [" + strings.Join(item.Tags, ",") + "]"
		}
		urgent := ""
		if item.Urgent {
			urgent = " !"
		}
		fmt.Fprintf(w, "%s\t%s%s%s\t%s%s\n",
			item.ID, prefix, done, urgent, item.Text, tags)
	}
	w.Flush()
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "odak:", err)
		os.Exit(1)
	}
}
