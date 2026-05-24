// Package mcp implements a Model Context Protocol server over stdio (JSON-RPC 2.0).
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

func respond(enc *json.Encoder, id any, result any) {
	enc.Encode(msg{JSONRPC: "2.0", ID: id, Result: result})
}

func errResp(enc *json.Encoder, id any, code int, text string) {
	enc.Encode(msg{JSONRPC: "2.0", ID: id, Error: &rpcErr{Code: code, Message: text}})
}

// tool descriptor for initialize response
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

var tools = []toolDef{
	{
		Name:        "list_todos",
		Description: "List todo items, optionally filtered by section or tag.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"section": map[string]any{"type": "string", "description": "Filter by section name (e.g. Focus, Inbox)"},
				"tag":     map[string]any{"type": "string", "description": "Filter by tag"},
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
				"text":     map[string]any{"type": "string", "description": "Todo text"},
				"section":  map[string]any{"type": "string", "description": "Section to add to (default: Inbox)"},
				"urgent":   map[string]any{"type": "boolean", "description": "Mark as urgent"},
				"deadline": map[string]any{"type": "string", "description": "Deadline date (YYYY-MM-DD)"},
				"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tags"},
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
		Name:        "list_sections",
		Description: "List all sections with their item counts.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	},
}

func text(s string) any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": s}}}
}

func jsonText(v any) any {
	b, _ := json.MarshalIndent(v, "", "  ")
	return text(string(b))
}

func handle(enc *json.Encoder, c *client.Client, req msg) {
	params := map[string]any{}
	if len(req.Params) > 0 {
		// params may be {"arguments": {...}} (tools/call) or flat
		var wrap struct {
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &wrap); err == nil && wrap.Arguments != nil {
			params = wrap.Arguments
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

	switch req.Method {
	case "initialize":
		respond(enc, req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "odak", "version": "1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		})

	case "notifications/initialized":
		// no response needed

	case "tools/list":
		respond(enc, req.ID, map[string]any{"tools": tools})

	case "tools/call":
		name := str("name")
		// re-parse params to get arguments separately
		var callParams struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if len(req.Params) > 0 {
			json.Unmarshal(req.Params, &callParams)
			name = callParams.Name
			if callParams.Arguments != nil {
				params = callParams.Arguments
			}
		}

		switch name {
		case "list_todos":
			items, err := c.List(str("section"), str("tag"), "")
			if err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, jsonText(items))

		case "add_todo":
			item := &model.Item{
				Text:     str("text"),
				Section:  str("section"),
				Deadline: str("deadline"),
			}
			if item.Section == "" {
				item.Section = "Inbox"
			}
			if u, ok := params["urgent"].(bool); ok {
				item.Urgent = u
			}
			if tags, ok := params["tags"].([]any); ok {
				for _, t := range tags {
					if s, ok := t.(string); ok {
						item.Tags = append(item.Tags, s)
					}
				}
			}
			created, err := c.Create(item)
			if err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, jsonText(created))

		case "toggle_done":
			item, err := c.ToggleDone(str("id"))
			if err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, jsonText(item))

		case "delete_todo":
			if err := c.Delete(str("id")); err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, text("deleted"))

		case "move_todo":
			item, err := c.Move(str("id"), str("section"))
			if err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, jsonText(item))

		case "list_sections":
			sections, err := c.Sections()
			if err != nil {
				errResp(enc, req.ID, -32000, err.Error())
				return
			}
			respond(enc, req.ID, jsonText(sections))

		default:
			errResp(enc, req.ID, -32601, fmt.Sprintf("unknown tool: %s", name))
		}

	default:
		if req.ID != nil {
			errResp(enc, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
	}
}

func Run(c *client.Client) {
	enc := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req msg
		if err := json.Unmarshal(line, &req); err != nil {
			errResp(enc, nil, -32700, "parse error")
			continue
		}
		handle(enc, c, req)
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "odak mcp:", err)
	}
}
