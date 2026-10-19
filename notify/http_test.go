package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

//
//  code、Telegram ok ），**HTTP ** doJSON ，

func replyServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDoJSONClassifiesHTTPStatus(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		permanent bool
	}{
		{"200 ", 200, false},
		{"429 ", 429, false},
		{"408 ", 408, false},
		{"500 ", 500, false},
		{"502 ", 502, false},
		{"503 ", 503, false},
		{"400 ", 400, true},
		{"401 ", 401, true},
		{"403 ", 403, true},
		{"404 ", 404, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := replyServer(t, tc.status, `{"detail":"upstream says no"}`)
			_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
			if tc.status < 300 {
				if err != nil {
					t.Fatalf("2xx : %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal(" 2xx ")
			}
			if got := IsPermanent(err); got != tc.permanent {
				t.Fatalf("HTTP %d  permanent ： %v  %v (%v)",
					tc.status, tc.permanent, got, err)
			}
			//  Go  StatusText：
			if !strings.Contains(err.Error(), strconv.Itoa(tc.status)) {
				t.Errorf(" HTTP  %d， %v", tc.status, err)
			}
		})
	}
}

// TestDoJSONIncludesResponseSnippet  snippet：，
func TestDoJSONIncludesResponseSnippet(t *testing.T) {
	srv := replyServer(t, 400, `{"error":"invalid webhook token"}`)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("")
	}
	if !strings.Contains(err.Error(), "invalid webhook token") {
		t.Errorf("， %v", err)
	}
}

// TestDoJSONSnippetIsSingleLineAndBounded  snippet ：
func TestDoJSONSnippetIsSingleLineAndBounded(t *testing.T) {
	long := strings.Repeat("x", 5000)
	srv := replyServer(t, 500, "line1\nline2\r\n\tline3 "+long)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("")
	}
	msg := err.Error()
	if strings.ContainsAny(msg, "\r\n\t") {
		t.Errorf("， %q", msg)
	}
	if len(msg) > 400 {
		t.Errorf("（%d ）， snippet : %q", len(msg), msg)
	}
}

// TestDoJSONRejectsOversizedResponse ：
func TestDoJSONRejectsOversizedResponse(t *testing.T) {
	huge := strings.Repeat("A", 1<<20) // 1 MiB
	srv := replyServer(t, 400, huge)
	_, err := doJSON(context.Background(), "GET", srv.URL, nil, nil)
	if err == nil {
		t.Fatal("")
	}
	if len(err.Error()) > 400 {
		t.Errorf("， %d", len(err.Error()))
	}
}
