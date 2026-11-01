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
	markupTagRe      = regexp.MustCompile(`<[^>]*>`)
	jsonRecordRe     = regexp.MustCompile(`\{\s*"[^"]+"\s*:`)
	jsonRecordLeadRe = regexp.MustCompile(`^\s*\{\s*"[^"]+"\s*:`)
	quoteableCTRe    = regexp.MustCompile(`^(text/plain|text/html)?$`)
)

// redactedEndpointURL strips the query string and userinfo from the endpoint
// URL before it is embedded in an error message, mirroring the SDK's redaction
// posture (query parameters are a legitimate credential channel). The scheme,
// host and escaped path are kept for diagnosability.
func redactedEndpointURL(endpointUrl string) string {
	u, err := url.Parse(endpointUrl)
	if err != nil || !u.IsAbs() {
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
// snippet is suppressed when it is empty, JSON-record-shaped (a `{"key":`
// record, at any position — including leading), or contains an "@" (a strict
// superset of an email match, so an email split across the read boundary
// cannot leak). A leading `[` or `{` alone does NOT suppress: only the
// record rule above suppresses, so plain text like "[ERROR] upstream:
// timeout" is still quoted, matching the SDK's record-based posture.
func sanitizedBodySnippet(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return ""
	}
	// The SDK's quotedBody counts the FULL body size (it reads the entire
	// body), while this connector reads at most 4 KiB. When ContentLength is
	// known (> 0), report it so the omission count matches the SDK's number;
	// for unknown lengths (-1, e.g. chunked responses), fall back to the
	// bytes actually read.
	bodySize := len(body)
	if resp.ContentLength > 0 {
		bodySize = int(resp.ContentLength)
	}
	ct := strings.Split(resp.Header.Get("Content-Type"), ";")
	if !quoteableCTRe.MatchString(strings.ToLower(strings.TrimSpace(ct[0]))) {
		return " (body omitted (" + strconv.Itoa(bodySize) + " bytes))"
	}
	s := markupTagRe.ReplaceAllString(string(body), " ")
	s = html.UnescapeString(s)
	// Check for JSON-record shapes BEFORE the whitespace collapse, so records
	// embedded mid-text (e.g. `oops {"request_id": 42}`) are suppressed too.
	if jsonRecordRe.MatchString(s) {
		return ""
	}
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsPrint(r) && r != ' ' }), " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || jsonRecordLeadRe.MatchString(s) || strings.Contains(s, "@") {
		return ""
	}
	r := []rune(s)
	if len(r) > maxSnippetLength {
		s = string(r[:maxSnippetLength]) + "..."
	}
	return " (body: " + s + ")"
}
