package agent

// bedrock.go — AWS Bedrock integration via Signature V4 signing.
//
// Bedrock exposes Claude models on the Anthropic Messages API wire format at:
//   https://bedrock-runtime.<region>.amazonaws.com/model/<model-id>/invoke (non-streaming)
//   https://bedrock-runtime.<region>.amazonaws.com/model/<model-id>/invoke-with-response-stream (streaming)
//
// The norma SDK speaks to any Anthropic-compatible endpoint, so we reuse the
// existing FormatAnthropic provider and inject a custom http.RoundTripper that:
//   1. Rewrites the URL to the Bedrock endpoint for the requested model.
//   2. Strips the x-api-key / Authorization headers set by the Anthropic adapter.
//   3. Signs the request with AWS Signature V4 (signing service = "bedrock", endpoint prefix = "bedrock-runtime").
//
// API key field encoding (stored in llm_profiles.api_key):
//   "<accessKeyID>|<secretAccessKey>|<region>"
//   e.g. "AKIAIOSFODNN7EXAMPLE|wJalrXUtn/...|us-east-1"
//
// The model field stores the Bedrock model ID, e.g.:
//   "anthropic.claude-3-5-sonnet-20241022-v2:0"

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// bedrockEndpointPrefix is the hostname prefix for the Bedrock Runtime endpoint.
// bedrockSigningName is the AWS SigV4 signing service name — intentionally different
// from the endpoint prefix. AWS Bedrock Runtime uses "bedrock" as the signing name
// even though the endpoint hostname uses "bedrock-runtime".
const bedrockEndpointPrefix = "bedrock-runtime"
const bedrockSigningName = "bedrock"

// BedrockTransport is an http.RoundTripper that signs requests with AWS
// Signature V4 and rewrites URLs to the Bedrock Runtime endpoint.
type BedrockTransport struct {
	AccessKey  string
	SecretKey  string
	Region     string
	ModelID    string
	Inner      http.RoundTripper
}

// RoundTrip signs the request and forwards it to Bedrock.
func (t *BedrockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request so we don't mutate the original.
	clone := req.Clone(req.Context())

	// Read the body so we can include it in the signature.
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("bedrock: read body: %w", err)
		}
		req.Body.Close()
		clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		clone.ContentLength = int64(len(bodyBytes))
	}

	// Rewrite URL to Bedrock endpoint.
	// Detect streaming: the norma Anthropic adapter uses "/v1/messages" for stream
	// and sets "Accept: text/event-stream". Bedrock uses different paths.
	isStream := strings.Contains(req.Header.Get("Accept"), "text/event-stream")
	var path string
	if isStream {
		path = fmt.Sprintf("/model/%s/invoke-with-response-stream", t.ModelID)
	} else {
		path = fmt.Sprintf("/model/%s/invoke", t.ModelID)
	}
	clone.URL.Scheme = "https"
	clone.URL.Host = fmt.Sprintf("%s.%s.amazonaws.com", bedrockEndpointPrefix, t.Region)
	clone.URL.Path = path
	clone.URL.RawQuery = ""
	clone.Host = clone.URL.Host

	// Remove Anthropic-specific auth headers; Bedrock uses SigV4.
	clone.Header.Del("x-api-key")
	clone.Header.Del("Authorization")
	// Bedrock doesn't accept anthropic-version/anthropic-beta as HTTP headers —
	// instead it requires "anthropic_version" inside the JSON body.
	clone.Header.Del("anthropic-version")
	clone.Header.Del("anthropic-beta")

	// Inject "anthropic_version":"bedrock-2023-05-31" into the JSON body.
	// Bedrock's Anthropic API always requires this field in the request body.
	// Also strip fields Bedrock doesn't support: output_config, stream (non-streaming).
	if len(bodyBytes) > 0 {
		var bodyMap map[string]json.RawMessage
		if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
			// Inject anthropic_version if missing
			if _, hasVer := bodyMap["anthropic_version"]; !hasVer {
				bodyMap["anthropic_version"] = json.RawMessage(`"bedrock-2023-05-31"`)
			}
			// Remove fields Bedrock doesn't accept:
			// - model: Bedrock specifies model in URL path, rejects it in body
			// - stream: Bedrock uses URL path routing (/invoke vs /invoke-with-response-stream)
			// - output_config: Anthropic reasoning effort field, not in Bedrock API
			delete(bodyMap, "model")
			delete(bodyMap, "stream")
			delete(bodyMap, "output_config")
			// Remove thinking field if it's set to disabled (Bedrock rejects it)
			if thinkingRaw, ok := bodyMap["thinking"]; ok {
				var thinking struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(thinkingRaw, &thinking) == nil && thinking.Type == "disabled" {
					delete(bodyMap, "thinking")
				}
			}
			if newBody, err := json.Marshal(bodyMap); err == nil {
				bodyBytes = newBody
				clone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				clone.ContentLength = int64(len(bodyBytes))
			}
		}
	}

	// Ensure Content-Type.
	if clone.Header.Get("Content-Type") == "" {
		clone.Header.Set("Content-Type", "application/json")
	}

	// Sign with AWS Signature V4.
	now := time.Now().UTC()
	if err := signV4(clone, bodyBytes, t.AccessKey, t.SecretKey, t.Region, now); err != nil {
		return nil, fmt.Errorf("bedrock: sign: %w", err)
	}

	return t.Inner.RoundTrip(clone)
}

// ─── AWS Signature V4 ────────────────────────────────────────────────────────

// sigV4URIEncode encodes a URI path for use in an AWS SigV4 canonical request.
// SigV4 rules: only unreserved chars (A-Z a-z 0-9 - _ . ~) and forward slashes
// '/' are kept as-is; everything else is percent-encoded (%XX uppercase).
// This is stricter than RFC 3986 path encoding — e.g. ':' in Bedrock model IDs
// like "anthropic.claude-3-5-sonnet-20241022-v2:0" must become "%3A".
func sigV4URIEncode(path string) string {
	var buf strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' || c == '/' {
			buf.WriteByte(c)
		} else {
			fmt.Fprintf(&buf, "%%%02X", c)
		}
	}
	return buf.String()
}

func signV4(req *http.Request, body []byte, accessKey, secretKey, region string, t time.Time) error {
	dateStamp := t.Format("20060102")
	amzDate := t.Format("20060102T150405Z")

	// Set required headers before signing.
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("host", req.URL.Host)

	// Canonical request.
	payloadHash := sha256Hex(body)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	signedHeaders, canonicalHeaders := buildHeaders(req)
	// SigV4 canonical URI uses stricter encoding than RFC 3986 path rules:
	// only unreserved chars (A-Z a-z 0-9 - _ . ~) and '/' are kept as-is;
	// everything else (including ':' in Bedrock model IDs) is %XX-encoded.
	// Go's url.EscapedPath() follows RFC 3986 which allows ':' in path segments,
	// so we must re-encode the path ourselves to match what AWS verifies against.
	canonicalURI := sigV4URIEncode(req.URL.Path)
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQueryString := req.URL.RawQuery

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign.
	credentialScope := strings.Join([]string{dateStamp, region, bedrockSigningName, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Signing key.
	signingKey := deriveSigningKey(secretKey, dateStamp, region)

	// Signature.
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Authorization header.
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature,
	))
	return nil
}

// buildHeaders returns (signedHeaders, canonicalHeaders) for the request.
// Only headers that should be signed (host, content-type, x-amz-*) are included.
func buildHeaders(req *http.Request) (string, string) {
	type kv struct{ k, v string }
	var pairs []kv

	// Always include host.
	pairs = append(pairs, kv{"host", req.URL.Host})

	// Include content-type and all x-amz-* headers.
	for k, vv := range req.Header {
		lk := strings.ToLower(k)
		if lk == "content-type" || strings.HasPrefix(lk, "x-amz-") {
			pairs = append(pairs, kv{lk, strings.Join(vv, ",")})
		}
	}

	// Sort by key name (required by SigV4).
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].k < pairs[j-1].k; j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}

	var headerLines []string
	var headerNames []string
	for _, p := range pairs {
		headerLines = append(headerLines, p.k+":"+strings.TrimSpace(p.v))
		headerNames = append(headerNames, p.k)
	}

	return strings.Join(headerNames, ";"), strings.Join(headerLines, "\n") + "\n"
}

func deriveSigningKey(secretKey, dateStamp, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(bedrockSigningName))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ParseBedrockAPIKey splits the encoded api_key "<accessKey>|<secretKey>|<region>"
// into its three components. Returns an error if the format is wrong.
func ParseBedrockAPIKey(apiKey string) (accessKey, secretKey, region string, err error) {
	parts := strings.SplitN(apiKey, "|", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("bedrock: api_key must be 'accessKey|secretKey|region', got %q", apiKey)
	}
	return parts[0], parts[1], parts[2], nil
}
