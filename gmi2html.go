package main

import (
	"fmt"
	"html"
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
				fmt.Fprint(&b, "<ul>")
			}
		} else if list {
			list = false
			fmt.Fprint(&b, "</ul>")
		}
		switch l.(type) {
		case gemini.LineLink:
			link := l.(gemini.LineLink)
			url := html.EscapeString(link.URL)
			name := html.EscapeString(link.Name)
			if name == "" {
				name = url
			}
			fmt.Fprintf(&b, "<a href='%s'>%s</a>\n", url, name)
		case gemini.LinePreformattingToggle:
			pre = !pre
			if pre {
				altText := string(l.(gemini.LinePreformattingToggle))
				if altText != "" {
					altText = html.EscapeString(altText)
					fmt.Fprintf(&b, "<pre title='%s'>", altText)
				} else {
					fmt.Fprint(&b, "<pre>\n")
				}
			} else {
				fmt.Fprint(&b, "</pre>")
			}
		case gemini.LinePreformattedText:
			text := string(l.(gemini.LinePreformattedText))
			fmt.Fprintf(&b, "%s\n", html.EscapeString(text))
		case gemini.LineHeading1:
			text := string(l.(gemini.LineHeading1))
			fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(text))
		case gemini.LineHeading2:
			text := string(l.(gemini.LineHeading2))
			fmt.Fprintf(&b, "<h2>%s</h2>", html.EscapeString(text))
		case gemini.LineHeading3:
			text := string(l.(gemini.LineHeading3))
			fmt.Fprintf(&b, "<h3>%s</h3>", html.EscapeString(text))
		case gemini.LineListItem:
			text := string(l.(gemini.LineListItem))
			fmt.Fprintf(&b, "<li>%s</li>", html.EscapeString(text))
		case gemini.LineQuote:
			text := string(l.(gemini.LineQuote))
			fmt.Fprintf(&b, "<blockquote>%s</blockquote>", html.EscapeString(text))
		case gemini.LineText:
			text := string(l.(gemini.LineText))
			if text == "" {
				fmt.Fprint(&b, "\n")
			} else {
				fmt.Fprintf(&b, "%s\n", html.EscapeString(text))
			}
		}
	}
	if pre {
		fmt.Fprint(&b, "</pre>")
	}
	if list {
		fmt.Fprint(&b, "</ul>")
	}
	return b.String()
}
