package httputil

import (
	"strings"
	"testing"
)

func TestNormalizeCRLF_LiteralEscapes(t *testing.T) {
	input := `GET / HTTP/1.1\r\nHost: example.com\r\n\r\n`
	got := NormalizeCRLF(input)
	want := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeCRLF_AlreadyCorrect(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	got := NormalizeCRLF(input)
	if got != input {
		t.Fatalf("got %q, want %q", got, input)
	}
}

func TestNormalizeCRLF_BodyWithLiteralEscapes(t *testing.T) {
	// A raw request whose JSON body contains the literal four-character
	// sequence \r\n (valid JSON escape) must NOT have those bytes converted
	// to real CRLF. The input already has real CRLF in the headers, so
	// escape interpretation is skipped entirely.
	body := `{"m":"hello\r\nworld"}`
	input := "POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 22\r\n\r\n" + body
	got := NormalizeCRLF(input)
	if !strings.Contains(got, body) {
		t.Fatalf("NormalizeCRLF corrupted body with literal escapes:\n got %q", got)
	}
}

func TestNormalizeCRLF_MissingTrailing(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: example.com"
	got := NormalizeCRLF(input)
	want := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeCRLF_SingleTrailingCRLF(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: example.com\r\n"
	got := NormalizeCRLF(input)
	want := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeCRLF_BodyBytesArePreserved(t *testing.T) {
	body := "--b\r\nContent-Type: image/png\r\n\r\n\x89PNG\r\n\x1a\n\x00\x00\n--b--"
	input := "POST /upload HTTP/1.1\nHost: example.com\nContent-Length: 47\n\n" + body
	got := NormalizeCRLF(input)
	want := "POST /upload HTTP/1.1\r\nHost: example.com\r\nContent-Length: 47\r\n\r\n" + body
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeCRLF_BodyGetsNoTrailingTerminator(t *testing.T) {
	input := "POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\na=1"
	got := NormalizeCRLF(input)
	if got != input {
		t.Fatalf("got %q, want %q", got, input)
	}
}

func TestNormalizeCRLF_MixedHeaderLineEndings(t *testing.T) {
	input := "POST / HTTP/1.1\r\nHost: example.com\nContent-Length: 4\r\n\na\nb\n"
	got := NormalizeCRLF(input)
	want := "POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 4\r\n\r\na\nb\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNormalizeCRLF_LeadingBlankLineIsNotTheSeparator(t *testing.T) {
	input := "\r\nGET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	got := NormalizeCRLF(input)
	want := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseHostHeader_Found(t *testing.T) {
	raw := "GET / HTTP/1.1\nHost: example.com\nAccept: */*\n"
	got := ParseHostHeader(raw)
	if got != "example.com" {
		t.Fatalf("got %q, want %q", got, "example.com")
	}
}

func TestParseHostHeader_CaseInsensitive(t *testing.T) {
	raw := "GET / HTTP/1.1\nhost: EXAMPLE.COM\n"
	got := ParseHostHeader(raw)
	if got != "EXAMPLE.COM" {
		t.Fatalf("got %q, want %q", got, "EXAMPLE.COM")
	}
}

func TestParseHostHeader_NotFound(t *testing.T) {
	raw := "GET / HTTP/1.1\nAccept: */*\n"
	got := ParseHostHeader(raw)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestParseHostHeader_EmptyInput(t *testing.T) {
	got := ParseHostHeader("")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
