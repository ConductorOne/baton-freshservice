package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
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
	}{
		{
			name:     "GET 404 html body becomes actionable",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html; charset=utf-8",
			body:     "<html><body>404 page not found</body></html>",
			wantCode: codes.NotFound,
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
				"Retry-After":           "60000",
			},
			wantCode: codes.Unavailable,
			contains: []string{"429"},
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
			name:     "email in body is redacted",
			method:   http.MethodGet,
			status:   404,
			ct:       "text/html",
			body:     "<html>contact admin@example.com</html>",
			wantCode: codes.NotFound,
			absent:   []string{"admin@example.com", "body:"},
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
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer ts.Close()

			c := newTestClient(ts)
			err := doReq(t, c, tc.method, ts.URL+"/agents", tc.res)
			if err == nil {
				t.Fatalf("expected non-nil error")
			}
			if tc.wantCode != codes.OK && status.Code(err) != tc.wantCode {
				t.Fatalf("code = %v, want %v (err: %v)", status.Code(err), tc.wantCode, err)
			}
			if tc.exactMsg != "" {
				// The SDK prepends the bare status line to the wrapped error.
				if !strings.Contains(err.Error(), tc.exactMsg) {
					t.Fatalf("message = %q, want to contain exactly %q", err.Error(), tc.exactMsg)
				}
				return
			}
			for _, want := range tc.contains {
				if !contains(err.Error(), want) {
					t.Errorf("message %q missing %q", err.Error(), want)
				}
			}
			for _, absent := range tc.absent {
				if contains(err.Error(), absent) {
					t.Errorf("message %q must not contain %q", err.Error(), absent)
				}
			}
			if tc.wantCode == codes.Unavailable && tc.status == 429 {
				details := status.Convert(err).Details()
				if len(details) != 1 {
					t.Fatalf("expected exactly 1 detail, got %d", len(details))
				}
				rl, ok := details[0].(*v2.RateLimitDescription)
				if !ok {
					t.Fatalf("detail type = %T, want *v2.RateLimitDescription", details[0])
				}
				if rl.GetRemaining() != 0 || rl.GetLimit() != 100 {
					t.Errorf("rate limit detail = %+v, want remaining 0 limit 100", rl)
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && strings.Contains(s, sub)
}