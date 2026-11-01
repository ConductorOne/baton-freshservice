package client

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/status"
)

const maxSnippetLength = 256

var (
	markupTagRe = regexp.MustCompile(`<[^>]*>`)
	emailRe     = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

// actionableHTTPError converts non-JSON HTTP error responses (status >= 300)
// into an actionable gRPC status error carrying the HTTP method, endpoint URL,
// status code, content type and a sanitized body snippet. It returns nil when
// the predicate does not match (JSON error responses keep the SDK's error
// path, and a nil response leaves the original error untouched).
func actionableHTTPError(method, endpointUrl string, resp *http.Response) error {
	if resp == nil || resp.StatusCode < http.StatusMultipleChoices || uhttp.IsJSONContentType(resp.Header.Get("Content-Type")) {
		return nil
	}

	code := uhttp.GrpcCodeFromHTTPStatus(resp.StatusCode)
	msg := fmt.Sprintf("freshservice-connector: %s %s: HTTP %d with content type %q%s",
		method, endpointUrl, resp.StatusCode, resp.Header.Get("Content-Type"), sanitizedBodySnippet(resp))

	rl, rlErr := extractRateLimitData(resp)
	if rlErr == nil {
		st := status.New(code, msg)
		st2, dErr := st.WithDetails(rl)
		if dErr == nil {
			return st2.Err()
		}
	}
	return status.Error(code, msg)
}

// sanitizedBodySnippet reads the response body exactly once (bounded) and
// renders a single-line, sanitized snippet. It returns "" when the body is
// empty, JSON-shaped, contains an email address, or cannot be read.
func sanitizedBodySnippet(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return ""
	}
	s := markupTagRe.ReplaceAllString(string(body), " ")
	s = html.UnescapeString(s)
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsPrint(r) && r != ' ' }), " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") || emailRe.MatchString(s) {
		return ""
	}
	r := []rune(s)
	if len(r) > maxSnippetLength {
		s = string(r[:maxSnippetLength]) + "..."
	}
	return " (body: " + s + ")"
}