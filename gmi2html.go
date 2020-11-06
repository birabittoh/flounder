package main

import (
	"fmt"
	"html"
	"mime"
	neturl "net/url"
	"path"
	"strings"

	"git.sr.ht/~adnano/go-gemini"
)

func textToHTML(text gemini.Text) string {
	var b strings.Builder
	var pre bool
	var list bool
	for _, l := range text {
		if _, ok := l.(gemini.LineListItem); ok {
			if !list {
				list = true
				fmt.Fprint(&b, "<ul>\n")
			}
		} else if list {
			list = false
			fmt.Fprint(&b, "</ul>\n")
		}
		switch l.(type) {
		case gemini.LineLink:
			link := l.(gemini.LineLink)
			url := html.EscapeString(link.URL)
			name := html.EscapeString(link.Name)
			var internal bool
			parsedUrl, err := neturl.Parse(url) // simpler way to do this?
			if err == nil {
				// only show images inline if they are relative links
				internal = parsedUrl.Host == ""
			}
			if strings.HasPrefix(mime.TypeByExtension(path.Ext(url)), "image") && internal {
				fmt.Fprintf(&b, "<img src='%s' alt='%s'>\n", url, name)
			} else {
				if name == "" {
					name = url
				}
				fmt.Fprintf(&b, "<p><a href='%s'>%s</a></p>\n", url, name)
			}
		case gemini.LinePreformattingToggle:
			pre = !pre
			if pre {
				altText := string(l.(gemini.LinePreformattingToggle))
				if altText != "" {
					altText = html.EscapeString(altText)
					fmt.Fprintf(&b, "<pre title='%s'>\n", altText)
				}
				fmt.Fprint(&b, "<pre>\n")
			} else {
				fmt.Fprint(&b, "</pre>\n")
			}
		case gemini.LinePreformattedText:
			text := string(l.(gemini.LinePreformattedText))
			fmt.Fprintf(&b, "%s\n", html.EscapeString(text))
		case gemini.LineHeading1:
			text := string(l.(gemini.LineHeading1))
			fmt.Fprintf(&b, "<h1>%s</h1>\n", html.EscapeString(text))
		case gemini.LineHeading2:
			text := string(l.(gemini.LineHeading2))
			fmt.Fprintf(&b, "<h2>%s</h2>\n", html.EscapeString(text))
		case gemini.LineHeading3:
			text := string(l.(gemini.LineHeading3))
			fmt.Fprintf(&b, "<h3>%s</h3>\n", html.EscapeString(text))
		case gemini.LineListItem:
			text := string(l.(gemini.LineListItem))
			fmt.Fprintf(&b, "<li>%s</li>\n", html.EscapeString(text))
		case gemini.LineQuote:
			text := string(l.(gemini.LineQuote))
			fmt.Fprintf(&b, "<blockquote>%s</blockquote>\n", html.EscapeString(text))
		case gemini.LineText:
			text := string(l.(gemini.LineText))
			if text == "" {
				fmt.Fprint(&b, "<br>\n")
			} else {
				fmt.Fprintf(&b, "<p>%s</p>\n", html.EscapeString(text))
			}
		}
	}
	if pre {
		fmt.Fprint(&b, "</pre>\n")
	}
	if list {
		fmt.Fprint(&b, "</ul>\n")
	}
	return b.String()
}
