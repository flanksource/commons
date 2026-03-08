package har

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/clicky/api"
)

func statusStyle(code int) string {
	switch {
	case code >= 500:
		return "text-red-500 font-bold"
	case code >= 400:
		return "text-yellow-500 font-bold"
	case code >= 300:
		return "text-blue-500 font-bold"
	case code >= 200:
		return "text-green-500 font-bold"
	default:
		return "text-muted"
	}
}

func statusText(r Response) api.Text {
	return api.Text{}.
		AddText(fmt.Sprintf("%d", r.Status), statusStyle(r.Status)).
		AddText(" "+r.StatusText, statusStyle(r.Status))
}

const maxBodyDisplay = 4096

func formatBody(mimeType, text string) api.Code {
	if strings.Contains(mimeType, "json") {
		var raw json.RawMessage
		if json.Unmarshal([]byte(text), &raw) == nil {
			if indented, err := json.MarshalIndent(raw, "", "  "); err == nil {
				text = string(indented)
			}
		}
	}
	if len(text) > maxBodyDisplay {
		text = text[:maxBodyDisplay] + "\n... (truncated)"
	}
	return api.CodeBlock(mimeType, text)
}

func headersToDescriptionList(headers []Header) api.DescriptionList {
	items := make([]api.KeyValuePair, len(headers))
	for i, h := range headers {
		items[i] = api.KeyValuePair{Key: h.Name, Value: h.Value}
	}
	return api.DescriptionList{Items: items}
}

// Columns implements api.TableProvider.
func (e Entry) Columns() []api.ColumnDef {
	return []api.ColumnDef{
		api.Column("method").Label("Method").Style("font-bold text-green-500 uppercase").Build(),
		api.Column("url").Label("URL").MaxWidth(80).Build(),
		api.Column("status").Label("Status").Build(),
		api.Column("duration").Label("Duration").Build(),
		api.Column("size").Label("Size").Build(),
	}
}

// Row implements api.TableProvider.
func (e Entry) Row() map[string]any {
	row := map[string]any{
		"method": e.Request.Method,
		"url":    e.Request.URL,
		"status": statusText(e.Response),
	}

	dur := time.Duration(e.Time * float64(time.Millisecond))
	row["duration"] = api.Human(dur, "text-muted")

	if e.Response.Content.Size > 0 {
		row["size"] = api.HumanizeBytes(e.Response.Content.Size).Styles("text-muted")
	}

	return row
}

// RowDetail implements api.DetailProvider for expandable table rows.
func (e Entry) RowDetail() api.Textable {
	t := api.Text{}
	hasContent := false

	if len(e.Request.Headers) > 0 {
		hasContent = true
		t = t.AddText("Request Headers", "font-bold text-muted").NewLine().
			Add(headersToDescriptionList(e.Request.Headers))
	}

	if e.Request.PostData != nil && e.Request.PostData.Text != "" {
		if hasContent {
			t = t.NewLine()
		}
		hasContent = true
		t = t.AddText("Request Body", "font-bold text-muted").NewLine().
			Add(formatBody(e.Request.PostData.MimeType, e.Request.PostData.Text))
	}

	if len(e.Response.Headers) > 0 {
		if hasContent {
			t = t.NewLine()
		}
		hasContent = true
		t = t.AddText("Response Headers", "font-bold text-muted").NewLine().
			Add(headersToDescriptionList(e.Response.Headers))
	}

	if e.Response.Content.Text != "" {
		if hasContent {
			t = t.NewLine()
		}
		hasContent = true
		label := "Response Body"
		if e.Response.Content.Truncated {
			label += " (truncated)"
		}
		t = t.AddText(label, "font-bold text-muted").NewLine().
			Add(formatBody(e.Response.Content.MimeType, e.Response.Content.Text))
	}

	if !hasContent {
		return nil
	}

	return t
}

// Pretty returns a compact one-line summary.
func (e Entry) Pretty() api.Text {
	t := api.Text{}.
		AddText(e.Request.Method, "font-bold text-green-500 uppercase").
		AddText(" "+e.Request.URL, "font-bold").
		Add(statusText(e.Response).Prefix(" "))

	dur := time.Duration(e.Time * float64(time.Millisecond))
	t = t.AddText(" ").Add(api.Human(dur, "text-muted"))

	if e.Response.Content.Size > 0 {
		t = t.AddText(" ").Add(api.HumanizeBytes(e.Response.Content.Size).Styles("text-muted"))
	}

	return t
}

// Table returns a TextTable of all collected entries with row detail expansion.
func (c *Collector) Table() api.TextTable {
	return api.NewTableFrom(c.Entries())
}

// Pretty returns a compact summary of all collected entries.
func (c *Collector) Pretty() api.Text {
	entries := c.Entries()
	t := api.Text{}
	for i, e := range entries {
		if i > 0 {
			t = t.NewLine().AddText("───────────────────────────────────────", "text-muted").NewLine()
		}
		t = t.Add(e.Pretty())
	}
	return t
}
