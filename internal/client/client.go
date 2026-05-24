package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/zcag/odak/internal/model"
)

type Client struct {
	base   string
	token  string
	http   *http.Client
}

func New(endpoint, token string) *Client {
	return &Client{base: endpoint, token: token, http: &http.Client{}}
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

func decode[T any](resp *http.Response) (T, error) {
	var v T
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e map[string]string
		json.NewDecoder(resp.Body).Decode(&e)
		return v, fmt.Errorf("HTTP %d: %s", resp.StatusCode, e["error"])
	}
	return v, json.NewDecoder(resp.Body).Decode(&v)
}

func (c *Client) List(section, tag, parentID string) ([]*model.Item, error) {
	q := url.Values{}
	if section != "" {
		q.Set("section", section)
	}
	if tag != "" {
		q.Set("tag", tag)
	}
	if parentID != "" {
		q.Set("parent_id", parentID)
	}
	path := "/todos"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return decode[[]*model.Item](resp)
}

func (c *Client) Create(item *model.Item) (*model.Item, error) {
	resp, err := c.do("POST", "/todos", item)
	if err != nil {
		return nil, err
	}
	return decode[*model.Item](resp)
}

func (c *Client) Get(id string) (*model.Item, error) {
	resp, err := c.do("GET", "/todos/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decode[*model.Item](resp)
}

func (c *Client) Update(id string, patch *model.Item) (*model.Item, error) {
	resp, err := c.do("PATCH", "/todos/"+id, patch)
	if err != nil {
		return nil, err
	}
	return decode[*model.Item](resp)
}

func (c *Client) Delete(id string) error {
	resp, err := c.do("DELETE", "/todos/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ToggleDone(id string) (*model.Item, error) {
	resp, err := c.do("PATCH", "/todos/"+id+"/done", nil)
	if err != nil {
		return nil, err
	}
	return decode[*model.Item](resp)
}

func (c *Client) Move(id string, section model.Section) (*model.Item, error) {
	resp, err := c.do("POST", "/todos/"+id+"/move", map[string]string{"section": string(section)})
	if err != nil {
		return nil, err
	}
	return decode[*model.Item](resp)
}

type SectionInfo struct {
	Name  model.Section `json:"name"`
	Count int           `json:"count"`
}

func (c *Client) Sections() ([]SectionInfo, error) {
	resp, err := c.do("GET", "/sections", nil)
	if err != nil {
		return nil, err
	}
	return decode[[]SectionInfo](resp)
}

func (c *Client) GetRaw() (string, error) {
	resp, err := c.do("GET", "/raw", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

func (c *Client) PutRaw(content string) error {
	req, err := http.NewRequest("PUT", c.base+"/raw", bytes.NewBufferString(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
