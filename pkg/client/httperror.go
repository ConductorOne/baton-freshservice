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
	quoteableCTRe = regexp.MustCompile(`^(text/plain|text/html)?$`)
)

// sdkJsonRecordPattern is the vendored baton-sdk uhttp jsonRecord rule,
// ported verbatim from pkg/uhttp/wrapper.go: it matches an object or an
// array that quotes a string (or separates two single-quoted ones) — which
// is what a record does and a message does not — plus one that opens after
// a prefix such as an XSSI guard. "[ERROR] upstream: timeout" matches
// neither alternative.
const sdkJsonRecordPattern = `^[{\[].*(?:"|'\s*[:,])|[{\[]\s*"`

var (
	sdkJsonRecordRe = regexp.MustCompile(sdkJsonRecordPattern)
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
// path, and a nil response leaves the original error untouched), or when the
// original error is not the SDK's own classification of this response: if the
// error-body read itself failed mid-flight, the SDK classifies that as a
// transport error (Unavailable/Canceled) whose code differs from
// GrpcCodeFromHTTPStatus(resp.StatusCode); replacing it would yield a
// terminal-looking status and drop the only record of the cause, so the
// original error passes through untouched.
func actionableHTTPError(method, endpointUrl string, resp *http.Response, origErr error) error {
	if resp == nil || origErr == nil || resp.StatusCode < http.StatusMultipleChoices || uhttp.IsJSONContentType(resp.Header.Get("Content-Type")) {
		return nil
	}

	code := uhttp.GrpcCodeFromHTTPStatus(resp.StatusCode)
	// Fact-based guard: replace the SDK's error only when it is a gRPC status
	// error whose code equals the SDK's own classification of this response.
	// Any other error (e.g. a transport failure carrying the response) passes
	// through untouched.
	if status.Code(origErr) != code {
		return nil
	}
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

// sanitizedBodySnippet reads the response body exactly once and renders a
// single-line, sanitized snippet. Only plain-text bodies (absent, text/plain,
// text/html content types — the SDK's quotableMessage posture) are quoted;
// any other content type renders as a byte-count omission. The snippet is
// suppressed when it is empty, matches the SDK's record rule (an object or
// bracket-array that quotes a string, a single-quoted record, or a record
// opening after a prefix, at any position — see sdkJsonRecordPattern), or
// contains an "@" (a strict superset of an email match, so an email split
// across the read boundary cannot leak). A leading `[` or `{` alone does NOT
// suppress: only the record rule above suppresses, so plain text like
// "[ERROR] upstream: timeout" is still quoted, matching the SDK's
// record-based posture. A record-suppressed or non-quotable body renders
// "body omitted (N bytes)" with the FULL body size, matching the SDK's
// quotedBody count. The snippet itself is taken from at most the first
// 4 KiB of the body (that bound governs what gets quoted, not the count);
// the SDK's Do has already buffered the whole body into resp.Body, so
// counting costs one pass over in-memory bytes, never the network.
func sanitizedBodySnippet(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	bodySize := len(body)
	if resp.ContentLength > 0 {
		bodySize = int(resp.ContentLength)
	}
	omitted := " (body omitted (" + strconv.Itoa(bodySize) + " bytes))"
	ct := strings.Split(resp.Header.Get("Content-Type"), ";")
	if !quoteableCTRe.MatchString(strings.ToLower(strings.TrimSpace(ct[0]))) {
		return omitted
	}
	s := markupTagRe.ReplaceAllString(string(body), " ")
	s = html.UnescapeString(s)
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsPrint(r) && r != ' ' }), " ")
	s = strings.Join(strings.Fields(s), " ")
	// Check for record shapes AFTER the collapse, exactly as the SDK applies
	// jsonRecord to its single-line message: records embedded mid-text (e.g.
	// `oops {"request_id": 42}`) and leading records are both suppressed.
	if s == "" || sdkJsonRecordRe.MatchString(s) || strings.Contains(s, "@") {
		if sdkJsonRecordRe.MatchString(s) {
			return omitted
		}
		return ""
	}
	r := []rune(s)
	if len(r) > maxSnippetLength {
		s = string(r[:maxSnippetLength]) + "..."
	}
	return " (body: " + s + ")"
}
