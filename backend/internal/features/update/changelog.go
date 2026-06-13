package update

import (
	"bytes"
	"html"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var (
	changelogMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM))
	changelogPolicy   = bluemonday.UGCPolicy()
)

// RenderChangelogHTML converts a Markdown changelog into a sanitized, minimal
// standalone HTML document suitable for the release notes link.
func RenderChangelogHTML(title, markdown string) ([]byte, error) {
	var body bytes.Buffer
	if err := changelogMarkdown.Convert([]byte(markdown), &body); err != nil {
		return nil, err
	}
	safe := changelogPolicy.SanitizeBytes(body.Bytes())

	var doc bytes.Buffer
	doc.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\">")
	doc.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	doc.WriteString("<title>")
	doc.WriteString(html.EscapeString(title))
	doc.WriteString("</title><style>")
	doc.WriteString("body{font:14px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#1a1a1a;max-width:680px;margin:24px auto;padding:0 16px}")
	doc.WriteString("h1,h2,h3{line-height:1.25}code{background:#f3f3f3;padding:1px 4px;border-radius:4px}")
	doc.WriteString("pre{background:#f3f3f3;padding:12px;border-radius:8px;overflow:auto}a{color:#2563eb}")
	doc.WriteString("</style></head><body>")
	doc.Write(safe)
	doc.WriteString("</body></html>")
	return doc.Bytes(), nil
}
