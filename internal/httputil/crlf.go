package httputil

import "strings"

// NormalizeCRLF rewrites the request line and header lines of a raw
// HTTP request to CRLF line endings and terminates the header block with
// a blank line. The body, everything after the first blank line, is
// returned byte-for-byte: rewriting it would corrupt binary uploads and
// multipart parts and break the request's Content-Length.
func NormalizeCRLF(raw string) string {
	// Only interpret literal \r\n escapes when the input has no real CR
	// or LF bytes. This supports CLI usage where users type \r\n
	// literally, while protecting body content that legitimately contains
	// the literal sequence (e.g. JSON values with escaped
	// carriage-return/newline) when the input already has real CRLF.
	if !strings.ContainsRune(raw, '\r') && !strings.ContainsRune(raw, '\n') {
		raw = strings.ReplaceAll(raw, `\r\n`, "\r\n")
	}
	var lines []string
	for {
		line, rest, found := strings.Cut(raw, "\n")
		line = strings.TrimSuffix(line, "\r")
		if found && line == "" && len(lines) > 0 {
			return strings.Join(lines, "\r\n") + "\r\n\r\n" + rest
		}
		if line != "" {
			lines = append(lines, line)
		}
		if !found {
			return strings.Join(lines, "\r\n") + "\r\n\r\n"
		}
		raw = rest
	}
}

func ParseHostHeader(raw string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "host:") {
			return strings.TrimSpace(line[5:])
		}
	}
	return ""
}
