package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"git.sr.ht/~adnano/go-gemini"
)

type ConvertedGmiDoc struct {
	Content string
	Title   string
}

func textToHTML(reqUrl *url.URL, text gemini.Text) ConvertedGmiDoc {
	var b strings.Builder
	var pre bool
	var blockquote bool
	var list bool
	var title string
	for _, l := range text {
		if _, ok := l.(gemini.LineQuote); ok {
			if !blockquote {
				blockquote = true
				fmt.Fprintf(&b, "<blockquote>\n")
			}
		} else if blockquote {
			blockquote = false
			fmt.Fprintf(&b, "</blockquote>\n")
		}
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
			urlstring := html.EscapeString(link.URL)
			// u = ctx.URL.ResolveReference(u) ?
			u, err := url.Parse(urlstring)
			if err != nil {
				continue
			}
			if reqUrl != nil {
				u = reqUrl.ResolveReference(u)
			}
			if u.Scheme == "gemini" || (reqUrl != nil && u.Scheme == "") {
				if strings.HasSuffix(u.Host, c.Host) {
					u.Scheme = ""
					urlstring = html.EscapeString(u.String())
				} else {
					u.Path = fmt.Sprintf("/%s%s", u.Host, u.Path)
					u.Scheme = ""
					u.Host = "proxy." + c.Host
					urlstring = html.EscapeString(u.String())
				}
			}
			name := html.EscapeString(link.Name)
			if name == "" {
				name = urlstring
			}
			fmt.Fprintf(&b, "<p><a href='%s'>%s</a></p>\n", urlstring, name)
		case gemini.LinePreformattingToggle:
			pre = !pre
			if pre {
				altText := string(l.(gemini.LinePreformattingToggle))
				if altText != "" {
					altText = html.EscapeString(altText)
					fmt.Fprintf(&b, "<pre title='%s'>\n", altText)
				} else {
					fmt.Fprint(&b, "<pre>\n")
				}
			} else {
				fmt.Fprint(&b, "</pre>\n")
			}
		case gemini.LinePreformattedText:
			text := string(l.(gemini.LinePreformattedText))
			fmt.Fprintf(&b, "%s\n", html.EscapeString(text))
		case gemini.LineHeading1:
			text := string(l.(gemini.LineHeading1))
			fmt.Fprintf(&b, "<h1>%s</h1>\n", html.EscapeString(text))
			if title == "" {
				title = text
			} // TODO deal with repetition
		case gemini.LineHeading2:
			text := string(l.(gemini.LineHeading2))
			fmt.Fprintf(&b, "<h2>%s</h2>\n", html.EscapeString(text))
			if title == "" {
				title = text
			}
		case gemini.LineHeading3:
			text := string(l.(gemini.LineHeading3))
			fmt.Fprintf(&b, "<h3>%s</h3>\n", html.EscapeString(text))
			if title == "" {
				title = text
			}
		case gemini.LineListItem:
			text := string(l.(gemini.LineListItem))
			fmt.Fprintf(&b, "<li>%s</li>\n", html.EscapeString(text))
		case gemini.LineQuote:
			text := string(l.(gemini.LineQuote))
			fmt.Fprintf(&b, "<p>%s</p>\n", html.EscapeString(text))
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
	return ConvertedGmiDoc{
		b.String(),
		title,
	}
}
