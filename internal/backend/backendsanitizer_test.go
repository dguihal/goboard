package backend

import (
	"testing"
)

func TestSanitize_PlainText(t *testing.T) {
	if got := Sanitize("hello world"); got != "hello world" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestSanitize_StripControlChars(t *testing.T) {
	got := Sanitize("hello\x01world")
	if got != "helloworld" {
		t.Errorf("expected control chars stripped, got %q", got)
	}
}

func TestSanitize_AllowedTags(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"<b>bold</b>", "<b>bold</b>"},
		{"<i>italic</i>", "<i>italic</i>"},
		{"<s>strike</s>", "<s>strike</s>"},
		{"<u>under</u>", "<u>under</u>"},
		{"<em>em</em>", "<em>em</em>"},
		{"<tt>mono</tt>", "<tt>mono</tt>"},
	}
	for _, tc := range cases {
		if got := Sanitize(tc.input); got != tc.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitize_ForbiddenTagEscaped(t *testing.T) {
	got := Sanitize("<script>alert(1)</script>")
	if got == "<script>alert(1)</script>" {
		t.Error("expected <script> to be escaped or stripped, got it unchanged")
	}
}

func TestSanitize_AnchorWithHref(t *testing.T) {
	got := Sanitize(`<a href="https://example.com">link</a>`)
	if got != `<a href="https://example.com">link</a>` {
		t.Errorf("expected anchor with href preserved, got %q", got)
	}
}

func TestSanitize_AnchorDropsForbiddenAttr(t *testing.T) {
	got := Sanitize(`<a onclick="bad()">link</a>`)
	if got == `<a onclick="bad()">link</a>` {
		t.Error("expected onclick attribute to be removed")
	}
}

func TestSanitize_HTMLSpecialCharsEscaped(t *testing.T) {
	got := Sanitize("a < b & c > d")
	if got == "a < b & c > d" {
		t.Error("expected <, &, > to be escaped")
	}
}

func TestSanitizeAndValidate_ValidMessage(t *testing.T) {
	_, err := SanitizeAndValidate("hello world")
	if err != nil {
		t.Errorf("expected no error for valid message, got: %v", err)
	}
}

func TestSanitizeAndValidate_EmptyMessage(t *testing.T) {
	_, err := SanitizeAndValidate("")
	if err == nil {
		t.Error("expected error for empty message")
	}
}
