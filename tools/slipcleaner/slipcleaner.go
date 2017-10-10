package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hishboy/gocommons/lang"
	"golang.org/x/net/html"
)

func main() {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		fmt.Println(sanitize(text))
	}
}

// Copy / Pasted from backendhandler.go
// Should be imported but need refactoring for being able to link this code

func sanitize(input string) string {

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
				str := fmt.Sprintf("</%s>", tn)

				for s.Len() > 0 {
					tmp := s.Pop().(token)

					if tmp.tokenType == html.StartTagToken && tmp.tagName != tnStr {
						// Not a corresponding open tag : sanitize it and store it as text
						str = fmt.Sprintf("%s%s", sanitizeChars(tmp.txt), str)
					} else {
						// a text or a corresponding open tag, at it as is
						str = fmt.Sprintf("%s%s", tmp.txt, str)

						if tmp.tagName == tnStr {
							// and leave if it's a corresponding open tag
							break
						}
					}
				}

				s.Push(token{
					txt:       str,
					tokenType: html.TextToken})
			} else {
				s.Push(token{
					txt:       sanitizeChars(string(z.Raw())),
					tokenType: html.TextToken})
			}

		default:
			re := regexp.MustCompile("(?i)https?://[\\da-z\\.-]+(?:/[^\\s\"]*)*/?")
			raw := string(z.Raw())
			if matches := re.FindAllStringIndex(raw, -1); matches != nil {
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
