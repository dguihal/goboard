package backend

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/hishboy/gocommons/lang"
	"golang.org/x/net/html"
)

/******************************************************************
 *             Backend Sanitizer
 ******************************************************************/

// Sanitize is the entry point for the backend sanitizer
func Sanitize(input string) string {

	if len(input) == 0 {
		return ""
	}

	tmp := stripCtlFromUTF8(input)
	tmp = htmlEscape(tmp)
	return tmp
}

// Remove unwanted (control) characters
func stripCtlFromUTF8(str string) string {
	return strings.Map(func(r rune) rune {
		if r >= 32 && r != 127 {
			return r
		}
		return -1
	}, str)
}

// HTML escape some conflicting characters
func sanitizeChars(input string) string {
	tmp := strings.Replace(input, "&", "&amp;", -1)
	tmp = strings.Replace(tmp, "<", "&lt;", -1)
	return strings.Replace(tmp, ">", "&gt;", -1)
}

// Allowed tags dictionnary
var allowedTags = map[string]bool{
	"a":  true,
	"b":  true,
	"i":  true,
	"s":  true,
	"tt": true,
	"em": true,
	"u":  true,
}

// Allowed attributes for tag dictionnary
var allowedAttrForTags = map[string][]string{
	"a": []string{"href"},
}

type token struct {
	txt       string
	tagName   string
	tokenType html.TokenType
}

func htmlEscape(input string) string {

	s := lang.NewStack()
	tagCount := map[string]int{}

	z := html.NewTokenizer(strings.NewReader(input))
	urlRe := regexp.MustCompile("(?i)https?://[\\da-z\\.-]+(?::\\d+)?(?:/[^\\s\"]*)*/?")

L:
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			break L
		case tt == html.StartTagToken:
			tn, hasAttrs := z.TagName()
			tnStr := string(tn)

			// Tag belongs to allowed list
			if allowedTags[tnStr] {
				tagAttrsStr := ""

				// Tag attributes management
				if allowedAttrs := allowedAttrForTags[tnStr]; hasAttrs && allowedAttrs != nil {

					moreAttr := hasAttrs
					for moreAttr {
						var key, val []byte
						key, val, moreAttr = z.TagAttr()
						for _, allowedAttr := range allowedAttrs {
							if string(key) == allowedAttr {
								tagAttrsStr = fmt.Sprintf(" %s=\"%s\"", string(key), val)
							}
						}
					}
				}

				s.Push(token{
					txt:       fmt.Sprintf("<%s%s>", tn, tagAttrsStr),
					tagName:   tnStr,
					tokenType: html.StartTagToken})

				// if a key doesn't exists it's value is 0
				tagCount[tnStr] = tagCount[tnStr] + 1
			} else {
				s.Push(token{
					txt:       sanitizeChars(string(z.Raw())),
					tokenType: html.TextToken})
			}
		case tt == html.EndTagToken:
			tn, _ := z.TagName()
			tnStr := string(tn)

			if allowedTags[tnStr] && tagCount[tnStr] > 0 {
				endStr := fmt.Sprintf("</%s>", tn)

				var strs []string
				startTagFound := false
				for s.Len() > 0 {
					tmp := s.Pop().(token)

					if tmp.tokenType == html.StartTagToken && tmp.tagName != tnStr {
						// Not a corresponding open tag : sanitize it and store it as text
						strs = append([]string{sanitizeChars(tmp.txt)}, strs...)
					} else {
						// a text or a corresponding open tag, at it as is
						strs = append([]string{tmp.txt}, strs...)

						if tmp.tagName == tnStr {
							startTagFound = true
							strs = append(strs, endStr)
							// and leave if it's a corresponding open tag
							break
						}
					}
				}
				if !startTagFound {
					strs = append(strs, sanitizeChars(endStr))
				}

				//Use a string buffer to build the final string from slice
				var buffer bytes.Buffer
				for elt := range strs {
					buffer.WriteString(strs[elt])
				}

				s.Push(token{
					txt:       buffer.String(),
					tokenType: html.TextToken})
			} else {
				s.Push(token{
					txt:       sanitizeChars(string(z.Raw())),
					tokenType: html.TextToken})
			}

		default:
			raw := string(z.Raw())
			if matches := urlRe.FindAllStringIndex(raw, -1); matches != nil {
				start := 0
				for _, match := range matches {
					if start < match[0] {
						s.Push(token{
							txt:       sanitizeChars(raw[start:match[0]]),
							tokenType: html.TextToken})
					}
					var buffer bytes.Buffer
					buffer.WriteString("<a href=\"")
					buffer.WriteString(raw[match[0]:match[1]])
					buffer.WriteString("\">[url]</a>")

					s.Push(token{
						txt:       buffer.String(),
						tokenType: html.TextToken})

					start = match[1]
				}

				if start < (len(raw)) {
					s.Push(token{
						txt:       sanitizeChars(raw[start:len(raw)]),
						tokenType: html.TextToken})

				}
			} else {
				s.Push(token{
					txt:       sanitizeChars(raw),
					tokenType: html.TextToken})
			}
		}
	}

	str := ""
	for s.Len() > 0 {
		tmp := s.Pop().(token)

		if tmp.tokenType != html.TextToken {
			str = fmt.Sprintf("%s%s", sanitizeChars(tmp.txt), str)
		} else {
			str = fmt.Sprintf("%s%s", tmp.txt, str)
		}
	}
	return str
}
