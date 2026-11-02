package mail

import (
	"bytes"
	"embed"
	"encoding/json"
	htmltemplate "html/template"
	"net/url"
	"strings"
	texttemplate "text/template"

	"savvy-go/internal/mail/smtp"
)

// files holds the HTML layout shared by every email (layout/layout.html: header with the logo,
// card, footer), the components a message is made of (layout/components.html), the logo that goes
// inline (layout/logo.png, written by `npm run logos`), the content of each message
// (messages/<name>.txt: components in order, rendered once as HTML and once as plain text) and the
// texts, a file per language (locales/<lang>.json).
//
//go:embed layout messages locales
var files embed.FS

// textComponents are the components of the plain-text part: a button is its bare link, and the
// link under it goes, since the link is already there.
const textComponents = `{{define "greeting"}}{{.Greeting}}{{end}}` +
	`{{define "body"}}{{.Body}}{{end}}` +
	`{{define "button"}}{{.Link}}{{end}}` +
	`{{define "expiry"}}{{.Expiry}}{{end}}` +
	`{{define "link-fallback"}}{{end}}`

// fallbackLang is the language of a request for one that has no texts.
const fallbackLang = "en"

// logoCID is the Content-ID of the inline logo, referenced as cid:logo in the layout.
const logoCID = "logo"

// catalog is the texts of one language. A text is a text/template over fields; the layout escapes
// it for HTML once it is rendered.
type catalog struct {
	Ignore       string             `json:"ignore"`
	LinkFallback string             `json:"linkFallback"`
	SentBy       string             `json:"sentBy"`
	Months       []string           `json:"months"`
	Roles        map[string]string  `json:"roles"`
	Messages     map[string]content `json:"messages"`
}

// content is the text of one message.
type content struct {
	Subject  string `json:"subject"`
	Greeting string `json:"greeting"`
	Body     string `json:"body"`
	Button   string `json:"button"`
	Expiry   string `json:"expiry"` // empty when the link has no expiry to show
}

var (
	catalogs = loadCatalogs()
	logo     = mustRead("layout/logo.png")
	layouts  = loadLayouts()
)

// layout is the HTML and plain-text part of one message.
type layout struct {
	html *htmltemplate.Template
	text *texttemplate.Template
}

// loadLayouts parses the content of every message of the fallback language twice: into the HTML
// layout with its components, and over the plain-text components. A message without content fails
// at startup, not when it is first sent.
func loadLayouts() map[string]layout {
	html := htmltemplate.Must(htmltemplate.ParseFS(files, "layout/layout.html", "layout/components.html"))
	text := texttemplate.Must(texttemplate.New("components").Parse(textComponents))
	out := map[string]layout{}
	for name := range catalogs[fallbackLang].Messages {
		src := string(mustRead("messages/" + name + ".txt"))
		out[name] = layout{
			html: htmltemplate.Must(htmltemplate.Must(html.Clone()).New("content").Parse(src)),
			text: texttemplate.Must(texttemplate.Must(text.Clone()).New("content").Parse(src)),
		}
	}
	return out
}

func mustRead(name string) []byte {
	b, err := files.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}

func loadCatalogs() map[string]catalog {
	entries, err := files.ReadDir("locales")
	if err != nil {
		panic(err)
	}
	out := map[string]catalog{}
	for _, e := range entries {
		lang, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		var c catalog
		if err := json.Unmarshal(mustRead("locales/"+e.Name()), &c); err != nil {
			panic(e.Name() + ": " + err.Error())
		}
		out[lang] = c
	}
	if _, ok := out[fallbackLang]; !ok {
		panic("no texts for " + fallbackLang)
	}
	return out
}

// Lang is the language an email is written in for a requested one: "ru-RU" and " RU " are "ru",
// and one without texts is English.
func Lang(requested string) string {
	lang := strings.ToLower(strings.TrimSpace(requested))
	lang, _, _ = strings.Cut(lang, "-")
	lang, _, _ = strings.Cut(lang, "_")
	if _, ok := catalogs[lang]; ok {
		return lang
	}
	return fallbackLang
}

// view is what the layouts show.
type view struct {
	Lang, Title, Greeting, Body, Button, Link, Expiry, Ignore, LinkFallback, Footer string
}

// render builds message name in lang: subject, the plain-text part and the HTML part.
func render(lang, name, to, link string, f fields) smtp.Message {
	lang = Lang(lang)
	c := catalogs[lang]
	msg := c.Messages[name]
	role := f.Role
	if label, ok := c.Roles[role]; ok {
		role = label
	}
	data := map[string]string{
		"Name": oneLine(f.Name), "Inviter": oneLine(f.Inviter), "Space": oneLine(f.Space), "Role": oneLine(role),
	}
	if !f.Expires.IsZero() {
		data["Expires"] = c.date(f.Expires)
	}
	v := view{
		Lang:         lang,
		Title:        fill(msg.Subject, data),
		Greeting:     fill(msg.Greeting, data),
		Body:         fill(msg.Body, data),
		Button:       fill(msg.Button, data),
		Link:         link,
		Ignore:       c.Ignore,
		LinkFallback: c.LinkFallback,
		Footer:       footer(c, link),
	}
	if !f.Expires.IsZero() {
		v.Expiry = fill(msg.Expiry, data)
	}
	var plain, page bytes.Buffer
	must(layouts[name].text.ExecuteTemplate(&plain, "content", v))
	must(layouts[name].html.ExecuteTemplate(&page, "layout.html", v))
	return smtp.Message{
		To:      to,
		Subject: v.Title,
		Text:    paragraphs(plain.String(), v.Ignore) + "\n\n--\n" + v.Footer + "\n",
		HTML:    page.String(),
		Inline:  []smtp.Inline{{CID: logoCID, Name: "logo.png", Type: "image/png", Data: logo}},
	}
}

// paragraphs joins the non-empty lines of a plain-text content and then more, a blank line between
// each: a component that is empty in plain text leaves no gap.
func paragraphs(content string, more ...string) string {
	var out []string
	for _, line := range append(strings.Split(content, "\n"), more...) {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n\n")
}

// footer names the instance the email came from: the host of the link.
func footer(c catalog, link string) string {
	if u, err := url.Parse(link); err == nil && u.Host != "" {
		return c.SentBy + " · " + u.Host
	}
	return c.SentBy
}

// fill renders one text of a catalog.
func fill(text string, data map[string]string) string {
	if text == "" {
		return ""
	}
	t := texttemplate.Must(texttemplate.New("").Option("missingkey=error").Parse(text))
	var b bytes.Buffer
	must(t.Execute(&b, data))
	return b.String()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
