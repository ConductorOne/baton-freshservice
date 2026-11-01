package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestClient(ts *httptest.Server) *FreshServiceClient {
	uhttpClient := uhttp.NewBaseHttpClient(ts.Client())
	return NewClient(uhttpClient).WithBaseURL(ts.URL).WithBearerToken("test-token")
}

// doReq runs a doRequest against the test server, optionally with a result
// target so the 200-path exercises uhttp.WithResponse.
func doReq(t *testing.T, c *FreshServiceClient, method, url string, res interface{}) error {
	t.Helper()
	_, _, err := c.doRequest(context.Background(), method, url, res, nil)
	return err
}

func TestActionableHTTPErrors(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		status   int
		body     string
		ct       string
		headers  map[string]string
		wantCode codes.Code
		contains []string
		absent   []string
		exactMsg string
		res      interface{}
		endpoint string // overrides the default ts.URL+"/agents" request URL
		wantFull bool   // assert the full (redacted) endpoint URL appears
		wantRL   bool   // assert a *v2.RateLimitDescription detail
		rlStatus v2.RateLimitDescription_Status
		resetSec int // expected ResetAt ≈ now + resetSec seconds
	}{
		{
			name:     "GET 404 html body becomes actionable",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html; charset=utf-8",
			body:     "<html><body>404 page not found</body></html>",
			wantCode: codes.NotFound,
			wantFull: true,
			contains: []string{"GET", "/agents", "404", "text/html", "404 page not found", "freshservice-connector:"},
		},
		{
			name:     "GET 404 html empty body omits snippet",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html",
			body:     "",
			wantCode: codes.NotFound,
			contains: []string{"GET", "/agents", "404", "text/html"},
			absent:   []string{"body:"},
		},
		{
			name:     "GET 404 json keeps SDK message exactly",
			method:   http.MethodGet,
			status:   404,
			ct:       "application/json",
			body:     `{"message":"Agent not found"}`,
			wantCode: codes.NotFound,
			exactMsg: "Request failed with status 404: Error: Agent not found",
		},
		{
			name:     "GET 403 text/plain actionable",
			method:   http.MethodGet,
			status:   403,
			ct:       "text/plain",
			body:     "forbidden by WAF",
			wantCode: codes.PermissionDenied,
			contains: []string{"403", "forbidden by WAF"},
		},
		{
			name:   "GET 429 html carries rate limit detail",
			method: http.MethodGet,
			status: 429,
			ct:     "text/html",
			body:   "<html>too many requests</html>",
			headers: map[string]string{
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Total":     "100",
				"Retry-After":           "60",
			},
			wantCode: codes.Unavailable,
			contains: []string{"429"},
			wantRL:   true,
			rlStatus: v2.RateLimitDescription_STATUS_OVERLIMIT,
			resetSec: 60,
		},
		{
			name:     "GET 429 with no rate limit headers gets SDK defaults",
			method:   http.MethodGet,
			status:   429,
			ct:       "text/html",
			body:     "<html>slow down</html>",
			wantCode: codes.Unavailable,
			contains: []string{"429"},
			wantRL:   true,
			rlStatus: v2.RateLimitDescription_STATUS_OVERLIMIT,
			resetSec: 60,
		},
		{
			name:     "DELETE 404 html becomes actionable",
			method:   http.MethodDelete,
			status:   404,
			ct:       "text/html",
			body:     "<html><body>not found</body></html>",
			wantCode: codes.NotFound,
			contains: []string{"DELETE", "404", "text/html"},
		},
		{
			// DELETE passes no uhttp options, so the JSON error body is not
			// decoded: today's (and D6's) behavior is the bare status line.
			name:     "DELETE 404 json keeps bare status line",
			method:   http.MethodDelete,
			status:   404,
			ct:       "application/json",
			body:     `{"message":"nope"}`,
			wantCode: codes.NotFound,
			contains: []string{"404 Not Found"},
			absent:   []string{"nope"},
		},
		{
			name:     "GET 500 html maps to Unavailable",
			method:   http.MethodGet,
			status:   500,
			ct:       "text/html",
			body:     "<html>upstream exploded</html>",
			wantCode: codes.Unavailable,
			contains: []string{"500", "upstream exploded"},
		},
		{
			name:     "csv error body is omitted not quoted",
			method:   http.MethodGet,
			status:   403,
			ct:       "text/csv",
			body:     "name,email,phone\nJane Doe,jane@example.com,555-0100",
			wantCode: codes.PermissionDenied,
			contains: []string{"403", "body omitted"},
			absent:   []string{"Jane Doe", "jane@example.com", "555-0100", "body:"},
		},
		{
			// Email truncated mid-domain by the 4KiB read boundary AND inside the first
			// 256 runes: the old emailRe (no TLD match) would quote "admin@examp";
			// the '@' suppression removes the snippet entirely.
			name:     "email cut at the 4KiB read boundary is suppressed",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html",
			body:     "<html>contact admin@examp" + strings.Repeat("x", 4096-len("<html>contact admin@examp")),
			wantCode: codes.NotFound,
			absent:   []string{"admin@examp", "@", "body:"},
		},
		{
			name:     "email in body is redacted",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html",
			body:     "<html>contact admin@example.com</html>",
			wantCode: codes.NotFound,
			absent:   []string{"admin@example.com", "@", "body:"},
		},
		{
			name:     "mid-text JSON record in body is suppressed",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/plain",
			body:     `oops {"request_id": 42, "secret": "x"} while handling`,
			wantCode: codes.NotFound,
			absent:   []string{"request_id", "secret", "body:"},
		},
		{
			// The omission count must reflect the FULL body size (the SDK's
			// quotedBody counts the whole body), not the 4 KiB the connector
			// reads. Content-Length is pinned so resp.ContentLength is known.
			name:     "oversized csv body omits full byte count",
			method:   http.MethodGet,
			status:   403,
			ct:       "text/csv",
			body:     "name,email\n" + strings.Repeat("jane, jane@example.com\n", 400),
			wantCode: codes.PermissionDenied,
			contains: []string{"403", "body omitted", fmt.Sprintf("%d bytes)", len("name,email\n"+strings.Repeat("jane, jane@example.com\n", 400)))},
			absent:   []string{"jane@example.com", "4096 bytes", "body:"},
		},
		{
			name:     "plain-text leading-bracket body is quoted",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/plain",
			body:     "[ERROR] upstream: timeout",
			wantCode: codes.NotFound,
			contains: []string{"[ERROR] upstream: timeout"},
			absent:   []string{"body omitted"},
		},
		{
			name:     "leading JSON record body is suppressed",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/plain",
			body:     `{"error": "boom"} trailing text`,
			wantCode: codes.NotFound,
			absent:   []string{"boom", "trailing", "body:"},
		},
		{
			// The SDK's jsonRecord rule also suppresses a leading bracket-array
			// that quotes strings; the connector must match that posture.
			name:     "quoted bracket-array body is suppressed with full count",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/plain",
			body:     `["jane","555-0100"]`,
			wantCode: codes.NotFound,
			contains: []string{"body omitted (19 bytes)"},
			absent:   []string{"jane", "555-0100", "body:"},
		},
		{
			name:     "single-quoted record body is suppressed",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/plain",
			body:     `{'name': 'jane doe'}`,
			wantCode: codes.NotFound,
			contains: []string{"body omitted"},
			absent:   []string{"jane doe", "body:"},
		},
		{
			name:     "query string in endpoint URL is redacted",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html",
			body:     "<html>gone</html>",
			endpoint: "/agents?page=2&secret=1",
			wantCode: codes.NotFound,
			contains: []string{"GET", "/agents"},
			absent:   []string{"secret=1", "?", "page=2"},
		},
		{
			name:     "GET 200 text/html success path untouched",
			method:   http.MethodGet,
			status:   200,
			ct:       "text/html",
			body:     "<html>ok</html>",
			res:      map[string]any{},
			wantCode: codes.Unknown,
			contains: []string{"unsupported content type"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tc.headers {
					w.Header().Set(k, v)
				}
				w.Header().Set("Content-Type", tc.ct)
				w.Header().Set("Content-Length", strconv.Itoa(len(tc.body)))
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer ts.Close()

			ep := ts.URL + "/agents"
			if tc.endpoint != "" {
				ep = ts.URL + tc.endpoint
			}
			c := newTestClient(ts)
			err := doReq(t, c, tc.method, ep, tc.res)
			if err == nil {
				t.Fatalf("expected non-nil error")
			}
			if tc.wantCode != codes.OK && status.Code(err) != tc.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", status.Code(err), tc.wantCode, err)
			}
			msg := err.Error()
			if tc.exactMsg != "" {
				// The SDK prepends the bare status line to the wrapped error.
				if !strings.Contains(msg, tc.exactMsg) {
					t.Fatalf("message = %q, want to contain exactly %q", msg, tc.exactMsg)
				}
				return
			}
			for _, want := range tc.contains {
				if !strings.Contains(msg, want) {
					t.Errorf("message %q missing %q", msg, want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(msg, absent) {
					t.Errorf("message %q must not contain %q", msg, absent)
				}
			}
			if tc.wantFull && !strings.Contains(msg, ep) {
				t.Errorf("message %q missing full endpoint URL %q", msg, ep)
			}
			if tc.wantRL {
				details := status.Convert(err).Details()
				if len(details) != 1 {
					t.Fatalf("expected exactly 1 detail, got %d", len(details))
				}
				rl, ok := details[0].(*v2.RateLimitDescription)
				if !ok {
					t.Fatalf("detail type = %T, want *v2.RateLimitDescription", details[0])
				}
				if rl.GetStatus() != tc.rlStatus {
					t.Errorf("rate limit status = %v, want %v", rl.GetStatus(), tc.rlStatus)
				}
				if tc.resetSec != 0 {
					want := time.Now().Add(time.Duration(tc.resetSec) * time.Second)
					reset := rl.GetResetAt().AsTime()
					if reset.Before(want.Add(-30*time.Second)) || reset.After(want.Add(30*time.Second)) {
						t.Errorf("resetAt = %v, want within 30s of %v", reset, want)
					}
				}
			}
		})
	}
}

// TestChunkedBodyOmitsFullCount pins the unknown-length (ContentLength == -1)
// residual: the omission count must reflect the FULL body the SDK's Do
// already buffered, not the 4 KiB snippet window.
func TestChunkedBodyOmitsFullCount(t *testing.T) {
	body := "name,email\n" + strings.Repeat("jane, jane@example.com\n", 400) // 9004 bytes
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Content-Length: HTTP/1.1 forces a chunked response, so the
		// client-side resp.ContentLength is -1.
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	err := doReq(t, c, http.MethodGet, ts.URL+"/agents", nil)
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, fmt.Sprintf("body omitted (%d bytes)", len(body))) {
		t.Errorf("message %q missing full chunked-body count %d", msg, len(body))
	}
	if strings.Contains(msg, "4096 bytes") || strings.Contains(msg, "jane@example.com") {
		t.Errorf("message %q must not understate the count or leak the body", msg)
	}
}

// TestBodyReadFailurePreservesTransportCause pins the replacement guard: when
// the error-body read itself fails mid-flight (here the handler dies after a
// 404 text/html head), the SDK classifies that as a transport error whose
// gRPC code differs from GrpcCodeFromHTTPStatus(404); the connector must pass
// that cause through instead of replacing it with a terminal-looking NotFound.
func TestBodyReadFailurePreservesTransportCause(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html>404 page not fo"))
		panic("handler died mid-body")
	}))
	defer ts.Close()

	c := newTestClient(ts)
	err := doReq(t, c, http.MethodGet, ts.URL+"/agents", nil)
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
	if code := status.Code(err); code == codes.NotFound {
		t.Errorf("transport cause was replaced with NotFound: %v", err)
	}
	if strings.Contains(err.Error(), "freshservice-connector:") {
		t.Errorf("transport cause was replaced by the connector: %v", err)
	}
}
