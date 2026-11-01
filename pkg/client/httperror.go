package client

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/conductorone/baton-sdk/pkg/ratelimit"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/status"
)

const maxSnippetLength = 256

var (
	markupTagRe   = regexp.MustCompile(`<[^>]*>`)
	jsonRecordRe  = regexp.MustCompile(`(?m)^\s*\{\s*"[^"]+"\s*:`)
	quoteableCTRe = regexp.MustCompile(`^(text/plain|text/html)?$`)
)

// redactedEndpointURL strips the query string and userinfo from the endpoint
// URL before it is embedded in an error message, mirroring the SDK's redaction
// posture (query parameters are a legitimate credential channel). The scheme,
// host and escaped path are kept for diagnosability.
func redactedEndpointURL(endpointUrl string) string {
	u, err := url.Parse(endpointUrl)
	if err != nil {
		return endpointUrl
	}
	return u.Scheme + "://" + u.Host + u.EscapedPath()
}

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
		method, redactedEndpointURL(endpointUrl), resp.StatusCode, resp.Header.Get("Content-Type"), sanitizedBodySnippet(resp))

	// Use the SDK's extractor (not the local one) so the rate-limit detail on
	// this error path matches the SDK's JSON error path byte-for-byte: same
	// Retry-After parsing, same 429 OVERLIMIT coercion and 60s default.
	rl, rlErr := ratelimit.ExtractRateLimitData(resp.StatusCode, &resp.Header)
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
// renders a single-line, sanitized snippet. Only plain-text bodies (absent,
// text/plain, text/html content types — the SDK's quotableMessage posture)
// are quoted; any other content type renders as a byte-count omission. The
// snippet is suppressed when it is empty, JSON-record-shaped, or contains an
// "@" (a strict superset of an email match, so an email split across the read
// boundary cannot leak).
func sanitizedBodySnippet(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return ""
	}
	ct := strings.Split(resp.Header.Get("Content-Type"), ";")
	if !quoteableCTRe.MatchString(strings.ToLower(strings.TrimSpace(ct[0]))) {
		return " (body omitted (" + strconv.Itoa(len(body)) + " bytes))"
	}
	s := markupTagRe.ReplaceAllString(string(body), " ")
	s = html.UnescapeString(s)
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsPrint(r) && r != ' ' }), " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") || jsonRecordRe.MatchString(s) || strings.Contains(s, "@") {
		return ""
	}
	r := []rune(s)
	if len(r) > maxSnippetLength {
		s = string(r[:maxSnippetLength]) + "..."
	}
	return " (body: " + s + ")"
}
