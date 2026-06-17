// Package mcp implements a Model Context Protocol server (JSON-RPC 2.0) over
// the Streamable HTTP transport. It is mounted on the odak API server and talks
// to the store through a loopback API client, reusing all the REST handler logic.
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zcag/odak/internal/client"
	"github.com/zcag/odak/internal/model"
)

type msg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func result(id any, r any) *msg { return &msg{JSONRPC: "2.0", ID: id, Result: r} }

func errMsg(id any, code int, text string) *msg {
	return &msg{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: text}}
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

var tools = []toolDef{
	{
		Name:        "list_todos",
		Description: "List todo items, optionally filtered by section, tag, or parent_id.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"section":   map[string]any{"type": "string", "description": "Filter by section name (e.g. Focus, Inbox)"},
				"tag":       map[string]any{"type": "string", "description": "Filter by tag"},
				"parent_id": map[string]any{"type": "string", "description": "Filter by parent item ID (returns children only)"},
			},
		},
	},
	{
		Name:        "get_todo",
		Description: "Get a single todo item by ID.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Item ID"},
			},
		},
	},
	{
		Name:        "add_todo",
		Description: "Add a new todo item.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"text"},
			"properties": map[string]any{
				"text":      map[string]any{"type": "string", "description": "Todo text"},
				"section":   map[string]any{"type": "string", "description": "Section to add to (default: Inbox)"},
				"urgent":    map[string]any{"type": "boolean", "description": "Mark as urgent"},
				"deadline":  map[string]any{"type": "string", "description": "Deadline date (YYYY-MM-DD)"},
				"trigger":   map[string]any{"type": "string", "description": "Wait/trigger date (YYYY-MM-DD)"},
				"tags":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tags as bare names, without the t: prefix (e.g. personal, work, infra) — the t: is added on render"},
				"parent_id": map[string]any{"type": "string", "description": "Parent item ID (creates a sub-item)"},
			},
		},
	},
	{
		Name:        "edit_todo",
		Description: "Update fields of an existing todo item. Only provided fields are changed.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id":        map[string]any{"type": "string", "description": "Item ID"},
				"text":      map[string]any{"type": "string", "description": "New text"},
				"urgent":    map[string]any{"type": "boolean", "description": "Mark as urgent"},
				"deadline":  map[string]any{"type": "string", "description": "Deadline date (YYYY-MM-DD), empty string to clear"},
				"trigger":   map[string]any{"type": "string", "description": "Wait/trigger date (YYYY-MM-DD), empty string to clear"},
				"tags":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Replace tags list; bare names without the t: prefix (e.g. personal, work, infra)"},
				"parent_id": map[string]any{"type": "string", "description": "Re-parent to this item ID, empty string to make top-level"},
			},
		},
	},
	{
		Name:        "toggle_done",
		Description: "Toggle the done state of a todo item.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Item ID"},
			},
		},
	},
	{
		Name:        "delete_todo",
		Description: "Delete a todo item.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id"},
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "Item ID"},
			},
		},
	},
	{
		Name:        "move_todo",
		Description: "Move a todo item to a different section.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"id", "section"},
			"properties": map[string]any{
				"id":      map[string]any{"type": "string", "description": "Item ID"},
				"section": map[string]any{"type": "string", "description": "Target section name"},
			},
		},
	},
	{
		Name:        "reorder_todos",
		Description: "Reorder items within a section by providing the full ordered list of IDs.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"section", "ids"},
			"properties": map[string]any{
				"section": map[string]any{"type": "string", "description": "Section name"},
				"ids":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Item IDs in desired order"},
			},
		},
	},
	{
		Name:        "list_sections",
		Description: "List all sections with their item counts.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	},
	{
		Name:        "get_raw",
		Description: "Get the raw Markdown content of the todos file.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	},
	{
		Name:        "put_raw",
		Description: "Overwrite the entire todos file with raw Markdown content.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"content"},
			"properties": map[string]any{
				"content": map[string]any{"type": "string", "description": "Full Markdown content to write"},
			},
		},
	},
}

func text(s string) any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": s}}}
}

func jsonText(v any) any {
	b, _ := json.MarshalIndent(v, "", "  ")
	return text(string(b))
}

func strSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, a := range arr {
		if s, ok := a.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// tagSlice is strSlice for tags: it tolerates a leading "t:" (the render-time
// prefix) so callers passing "t:personal" don't produce "[t:t:personal]".
func tagSlice(v any) []string {
	tags := strSlice(v)
	for i, t := range tags {
		tags[i] = strings.TrimPrefix(t, "t:")
	}
	return tags
}

// handle processes one JSON-RPC request and returns the response, or nil for
// notifications (which get no reply).
func handle(c *client.Client, req msg) *msg {
	// parse params: tools/call wraps args in {"name":..., "arguments":{...}}
	var callParams struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	params := map[string]any{}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &callParams); err == nil && callParams.Arguments != nil {
			params = callParams.Arguments
		} else {
			json.Unmarshal(req.Params, &params)
		}
	}

	str := func(k string) string {
		if v, ok := params[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	has := func(k string) bool { _, ok := params[k]; return ok }

	switch req.Method {
	case "initialize":
		ver := "2025-06-18"
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(req.Params, &p) == nil && p.ProtocolVersion != "" {
			ver = p.ProtocolVersion // echo the client's requested version
		}
		return result(req.ID, map[string]any{
			"protocolVersion": ver,
			"serverInfo":      map[string]any{"name": "odak", "version": "1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		})

	case "notifications/initialized":
		return nil // no response

	case "tools/list":
		return result(req.ID, map[string]any{"tools": tools})

	case "tools/call":
		name := callParams.Name
		if name == "" {
			name = str("name")
		}

		switch name {
		case "list_todos":
			items, err := c.List(str("section"), str("tag"), str("parent_id"))
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(items))

		case "get_todo":
			item, err := c.Get(str("id"))
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(item))

		case "add_todo":
			item := &model.Item{
				Text:     str("text"),
				Section:  model.Section(str("section")),
				Deadline: str("deadline"),
				Trigger:  str("trigger"),
				ParentID: str("parent_id"),
			}
			if item.Section == "" {
				item.Section = "Inbox"
			}
			if u, ok := params["urgent"].(bool); ok {
				item.Urgent = u
			}
			item.Tags = tagSlice(params["tags"])
			created, err := c.Create(item)
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(created))

		case "edit_todo":
			patch := &model.Item{}
			if has("text") {
				patch.Text = str("text")
			}
			if has("deadline") {
				patch.Deadline = str("deadline")
			}
			if has("trigger") {
				patch.Trigger = str("trigger")
			}
			if has("parent_id") {
				patch.ParentID = str("parent_id")
			}
			if u, ok := params["urgent"].(bool); ok {
				patch.Urgent = u
			}
			if has("tags") {
				patch.Tags = tagSlice(params["tags"])
			}
			item, err := c.Update(str("id"), patch)
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(item))

		case "toggle_done":
			item, err := c.ToggleDone(str("id"))
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(item))

		case "delete_todo":
			if err := c.Delete(str("id")); err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, text("deleted"))

		case "move_todo":
			item, err := c.Move(str("id"), model.Section(str("section")))
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(item))

		case "reorder_todos":
			ids := strSlice(params["ids"])
			if err := c.Reorder(str("section"), ids); err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, text("reordered"))

		case "list_sections":
			sections, err := c.Sections()
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, jsonText(sections))

		case "get_raw":
			content, err := c.GetRaw()
			if err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, text(content))

		case "put_raw":
			if err := c.PutRaw(str("content")); err != nil {
				return errMsg(req.ID, -32000, err.Error())
			}
			return result(req.ID, text("ok"))

		default:
			return errMsg(req.ID, -32601, fmt.Sprintf("unknown tool: %s", name))
		}

	default:
		if req.ID != nil {
			return errMsg(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
		return nil
	}
}

// Handler serves the MCP Streamable HTTP transport. POST carries a JSON-RPC
// request and gets a JSON response (or 202 for notifications); odak has no
// server-initiated messages, so GET (the optional SSE stream) returns 405.
func Handler(c *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8*1024*1024))
		if err != nil {
			writeRPC(w, errMsg(nil, -32700, "read error"))
			return
		}
		var req msg
		if err := json.Unmarshal(body, &req); err != nil {
			writeRPC(w, errMsg(nil, -32700, "parse error"))
			return
		}
		resp := handle(c, req)
		if resp == nil {
			w.WriteHeader(http.StatusAccepted) // notification: no reply
			return
		}
		writeRPC(w, resp)
	}
}

func writeRPC(w http.ResponseWriter, m *msg) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}
