package tools

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/c0tton-fluff/caido-mcp-server/v4/internal/replay"
	"github.com/c0tton-fluff/caido-mcp-server/v4/internal/testutil"
	caido "github.com/caido-community/sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestEditRequestReplaceMethod(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		newMethod string
		want      string
	}{
		{
			name:      "three token request line",
			raw:       "GET /api HTTP/1.1\r\nHost: example.com\r\n\r\n",
			newMethod: "POST",
			want:      "POST /api HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
		{
			name:      "two token request line",
			raw:       "GET /api\r\nHost: example.com\r\n\r\n",
			newMethod: "POST",
			want:      "POST /api\r\nHost: example.com\r\n\r\n",
		},
		{
			name:      "no CRLF returns input unchanged",
			raw:       "GET /api HTTP/1.1",
			newMethod: "POST",
			want:      "GET /api HTTP/1.1",
		},
		{
			name:      "single token request line unchanged",
			raw:       "GET\r\nHost: example.com\r\n\r\n",
			newMethod: "POST",
			want:      "GET\r\nHost: example.com\r\n\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replaceMethod(tt.raw, tt.newMethod); got != tt.want {
				t.Errorf("replaceMethod() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditRequestReplacePath(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		newPath string
		want    string
	}{
		{
			name:    "three token request line",
			raw:     "GET /old HTTP/1.1\r\nHost: example.com\r\n\r\n",
			newPath: "/new",
			want:    "GET /new HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
		{
			name:    "two token request line",
			raw:     "GET /old\r\nHost: example.com\r\n\r\n",
			newPath: "/new",
			want:    "GET /new\r\nHost: example.com\r\n\r\n",
		},
		{
			name:    "no CRLF returns input unchanged",
			raw:     "GET /old HTTP/1.1",
			newPath: "/new",
			want:    "GET /old HTTP/1.1",
		},
		{
			name:    "single token request line unchanged",
			raw:     "GET\r\nHost: example.com\r\n\r\n",
			newPath: "/new",
			want:    "GET\r\nHost: example.com\r\n\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replacePath(tt.raw, tt.newPath); got != tt.want {
				t.Errorf("replacePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditRequestRemoveHeader(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		hdr  string
		want string
	}{
		{
			name: "removes matching header preserves body",
			raw:  "GET / HTTP/1.1\r\nX-A: 1\r\nX-B: 2\r\n\r\nbody",
			hdr:  "X-A",
			want: "GET / HTTP/1.1\r\nX-B: 2\r\n\r\nbody",
		},
		{
			name: "case insensitive removal",
			raw:  "GET / HTTP/1.1\r\nContent-Type: text/html\r\nX-Custom: v\r\n\r\n",
			hdr:  "content-type",
			want: "GET / HTTP/1.1\r\nX-Custom: v\r\n\r\n",
		},
		{
			name: "colon-less header line preserved",
			raw:  "GET / HTTP/1.1\r\nBadHeaderNoColon\r\nHost: example.com\r\n\r\nbody",
			hdr:  "Host",
			want: "GET / HTTP/1.1\r\nBadHeaderNoColon\r\n\r\nbody",
		},
		{
			name: "no header block returns unchanged",
			raw:  "GET / HTTP/1.1\r\nHost: example.com",
			hdr:  "Host",
			want: "GET / HTTP/1.1\r\nHost: example.com",
		},
		{
			name: "absent header is no-op",
			raw:  "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
			hdr:  "X-Absent",
			want: "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeHeader(tt.raw, tt.hdr); got != tt.want {
				t.Errorf("removeHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestEditRequestReplaceBody pins the behavior of replaceBody.
//
// The header-body separator ("\r\n\r\n") must appear exactly once between
// the last header line and the body. Content-Length must equal len(newBody).
func TestEditRequestReplaceBody(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		newBody string
		want    string
		checkCL bool
	}{
		{
			name:    "injects Content-Length and replaces body",
			raw:     "POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello",
			newBody: "abc",
			want:    "POST /submit HTTP/1.1\r\nHost: example.com\r\nContent-Length: 3\r\n\r\nabc",
			checkCL: true,
		},
		{
			name:    "no header block appends separator and body",
			raw:     "GET / HTTP/1.1\r\nHost: example.com",
			newBody: "abc",
			want:    "GET / HTTP/1.1\r\nHost: example.com\r\n\r\nabc",
		},
		{
			name:    "empty body strips Content-Length only",
			raw:     "POST / HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello",
			newBody: "",
			want:    "POST / HTTP/1.1\r\nHost: example.com\r\n\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceBody(tt.raw, tt.newBody)
			if got != tt.want {
				t.Errorf("replaceBody() = %q, want %q", got, tt.want)
			}
			if tt.checkCL {
				wantHdr := "Content-Length: " + strconv.Itoa(len(tt.newBody))
				if !strings.Contains(got, wantHdr) {
					t.Errorf("replaceBody() missing %q in %q", wantHdr, got)
				}
			}
		})
	}
}

// TestEditRequestHandler exercises editRequestHandler end-to-end against the
// GraphQL mock: it fetches the source request, applies a method + header +
// body override, then runs the Caido 0.57 default-session send/poll flow
// (mirroring send_request_test.go's setupSendMocks wiring).
func TestEditRequestHandler(t *testing.T) {
	replay.ResetDefaultSession("")
	t.Cleanup(func() { replay.ResetDefaultSession("") })

	env := testutil.NewMCPTestEnv(t, func(s *mcp.Server, c *caido.Client) {
		RegisterEditRequestTool(s, c)
	})

	const (
		sessionID = "edit-sess"
		entryID   = "edit-entry"
		requestID = "edit-req"
	)

	// 1. Fetch the source request to modify.
	env.Mock.On("GetRequest", testutil.GetRequestFullResponse("req-src", "original body"))
	// 2. Default-session send flow: empty session created, no active entry ->
	//    fallback seeded session, task started, poll finds the new entry.
	env.Mock.On("CreateReplaySession", testutil.CreateReplaySessionResponse(sessionID))
	env.Mock.On("GetReplaySession", testutil.GetReplaySessionResponse(sessionID, ""))
	env.Mock.On("CreateReplaySession", testutil.CreateReplaySessionSeededResponse(sessionID, entryID))
	env.Mock.On("StartReplayTask", testutil.StartReplayTaskResponse())
	env.Mock.On("GetReplaySession", testutil.GetReplaySessionResponse(sessionID, entryID))
	env.Mock.On("GetReplayEntry", testutil.GetReplayEntryResponse(entryID, requestID, 200, "response body"))

	result := env.CallTool(t, "caido_edit_request", map[string]any{
		"requestId":  "req-src",
		"method":     "PUT",
		"setHeaders": map[string]any{"X-Test": "yes"},
		"body":       "newbody",
	})
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result.Content)
	}

	output := testutil.UnmarshalToolResult[SendRequestOutput](t, result)
	if output.SessionID != sessionID {
		t.Errorf("SessionID = %q, want %q", output.SessionID, sessionID)
	}
	if output.EntryID != entryID {
		t.Errorf("EntryID = %q, want %q", output.EntryID, entryID)
	}
	if output.RequestID != requestID {
		t.Errorf("RequestID = %q, want %q", output.RequestID, requestID)
	}
	if output.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", output.StatusCode)
	}
}

// TestEditRequestRawFile sends a request read from rawFile and checks the
// exact bytes and connection written to the replay draft.
func TestEditRequestRawFile(t *testing.T) {
	mpBody := "--b\r\nContent-Disposition: form-data; name=\"f\"\r\n\r\nline1\nline2\r\n--b--\r\n"
	multipart := "POST /upload HTTP/1.1\r\nHost: example.com\r\nCookie: sid=abc\r\nX-Trace: 1\r\n" +
		"Content-Type: multipart/form-data; boundary=b\r\nContent-Length: 69\r\n\r\n" + mpBody
	binBody := "\x89PNG\r\n\x1a\n\x00\xff\xfe\n"
	binary := "PUT /bin HTTP/1.1\r\nHost: example.com\r\n" +
		"Content-Type: application/octet-stream\r\nContent-Length: 12\r\n\r\n" + binBody

	tests := []struct {
		name     string
		file     string
		args     map[string]any
		want     string
		wantPort float64
		wantTLS  bool
		wantErr  string
	}{
		{
			name:     "multipart CRLF body unchanged",
			file:     multipart,
			args:     map[string]any{"host": "example.com"},
			want:     multipart,
			wantPort: 443,
			wantTLS:  true,
		},
		{
			name:     "binary body unchanged",
			file:     binary,
			args:     map[string]any{"host": "example.com", "port": 8080, "tls": false},
			want:     binary,
			wantPort: 8080,
		},
		{
			name:     "host:port split",
			file:     binary,
			args:     map[string]any{"host": "example.com:8443"},
			want:     binary,
			wantPort: 8443,
			wantTLS:  true,
		},
		{
			name: "header edits keep multipart body",
			file: multipart,
			args: map[string]any{
				"host":          "example.com",
				"setHeaders":    map[string]any{"Cookie": "sid=new"},
				"removeHeaders": []string{"X-Trace"},
			},
			want: "POST /upload HTTP/1.1\r\nHost: example.com\r\n" +
				"Content-Type: multipart/form-data; boundary=b\r\nContent-Length: 69\r\nCookie: sid=new\r\n\r\n" + mpBody,
			wantPort: 443,
			wantTLS:  true,
		},
		{
			name: "method path and body overrides",
			file: binary,
			args: map[string]any{
				"host": "example.com", "method": "POST", "path": "/new", "body": "a=1",
			},
			want: "POST /new HTTP/1.1\r\nHost: example.com\r\n" +
				"Content-Type: application/octet-stream\r\nContent-Length: 3\r\n\r\na=1",
			wantPort: 443,
			wantTLS:  true,
		},
		{
			name:    "missing host",
			file:    multipart,
			args:    map[string]any{},
			wantErr: "host is required with rawFile",
		},
		{
			name:    "requestId with rawFile",
			file:    multipart,
			args:    map[string]any{"host": "example.com", "requestId": "req-src"},
			wantErr: "exactly one of requestId or rawFile is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewMCPTestEnv(t, func(s *mcp.Server, c *caido.Client) {
				RegisterEditRequestTool(s, c)
			})
			env.Mock.On("GetReplaySession", testutil.GetReplaySessionResponse("raw-sess", "prev-entry"))
			env.Mock.On("UpdateReplayEntryDraft", testutil.UpdateReplayEntryDraftResponse("prev-entry"))
			env.Mock.On("StartReplayTask", testutil.StartReplayTaskResponse())
			env.Mock.On("GetReplaySession", testutil.GetReplaySessionResponse("raw-sess", "raw-entry"))
			env.Mock.On("GetReplayEntry", testutil.GetReplayEntryResponse("raw-entry", "raw-req", 200, "ok"))

			path := filepath.Join(t.TempDir(), "request.http")
			if err := os.WriteFile(path, []byte(tt.file), 0o600); err != nil {
				t.Fatal(err)
			}
			tt.args["rawFile"] = path
			tt.args["sessionId"] = "raw-sess"

			result := env.CallTool(t, "caido_edit_request", tt.args)
			if tt.wantErr != "" {
				text, _ := result.Content[0].(*mcp.TextContent)
				if !result.IsError || text == nil || !strings.Contains(text.Text, tt.wantErr) {
					t.Fatalf("want error %q, got %+v", tt.wantErr, result.Content)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected error result: %+v", result.Content)
			}

			draft := env.Mock.LastVariables("UpdateReplayEntryDraft")["input"].(map[string]any)["http"].(map[string]any)
			sent, err := base64.StdEncoding.DecodeString(draft["raw"].(string))
			if err != nil {
				t.Fatal(err)
			}
			if string(sent) != tt.want {
				t.Errorf("sent raw = %q, want %q", sent, tt.want)
			}
			conn := draft["connection"].(map[string]any)
			if conn["host"] != "example.com" || conn["port"] != tt.wantPort || conn["isTLS"] != tt.wantTLS {
				t.Errorf("connection = %v, want example.com:%v tls=%v", conn, tt.wantPort, tt.wantTLS)
			}
		})
	}
}
