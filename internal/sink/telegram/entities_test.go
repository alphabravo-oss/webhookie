package telegram

import "testing"

func TestParseHTML(t *testing.T) {
	ok := []string{
		"hello",
		"<b>bold</b>",
		"<b>bold <i>italic</i></b>",
		`<a href="https://example.com">x</a>`,
		"a &amp; b",
		"&#39;",
		"<blockquote expandable>q</blockquote>",
		`<span class="tg-spoiler">hide</span>`,
	}
	for _, s := range ok {
		if err := parseHTML(s); err != nil {
			t.Errorf("html %q: %v", s, err)
		}
	}
	bad := []string{
		"<b>bold",
		"<script>x</script>",
		"a < b",
		"a & b",
		"<b>bold</i>",
		"<a>no href</a>",
	}
	for _, s := range bad {
		if err := parseHTML(s); err == nil {
			t.Errorf("html %q: expected error", s)
		}
	}
}

func TestParseMarkdown(t *testing.T) {
	if err := parseMarkdown("*bold* _italic_ `code` [x](https://example.com)"); err != nil {
		t.Fatal(err)
	}
	if err := parseMarkdown("*unclosed"); err == nil {
		t.Fatal("expected unclosed")
	}
}

func TestParseMarkdownV2(t *testing.T) {
	ok := []string{
		"hello",
		`hello world\!`,
		"*bold*",
		"_italic_",
		"__underline__",
		"~strike~",
		"||spoiler||",
		`cost is \$5\.`,
		"[x](https://example.com)",
		"`code`",
		"```pre```",
		`*bold _italic_*`,
		">quoted line",
	}
	for _, s := range ok {
		if err := parseMarkdownV2(s); err != nil {
			t.Errorf("mdv2 %q: %v", s, err)
		}
	}
	bad := []string{
		"Hello world!",
		"file.txt",
		"*unclosed",
		"a (b) c",
		"#hashtag",
	}
	for _, s := range bad {
		if err := parseMarkdownV2(s); err == nil {
			t.Errorf("mdv2 %q: expected error", s)
		}
	}
}
