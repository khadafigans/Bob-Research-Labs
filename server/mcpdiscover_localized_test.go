package server

import (
	"context"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// mcpdiscover.go  connectMCP         
//   .   (stdio  ·http URL ·sse URL ·  
// connectMCP      .   F3a  
// assertKoreanError( ·  0) .   enum(stdio/http/sse)
func TestConnectMCPErrorsLocalized(t *testing.T) {
	for _, c := range []struct{ label, msg string }{
		{"stdio_no_command", errMCPStdioNoCommand},
		{"http_no_url", errMCPHTTPNoURL},
		{"sse_no_url", errMCPSSENoURL},
		{"unknown_transport", errMCPUnknownTransportFmt},
	} {
		assertKoreanError(t, c.label, c.msg)
	}

	for _, c := range []struct {
		label string
		srv   *db.MCPServer
	}{
		{"stdio_missing_command", &db.MCPServer{Transport: "stdio", Command: ""}},
		{"http_missing_url", &db.MCPServer{Transport: "http", URL: ""}},
		{"sse_missing_url", &db.MCPServer{Transport: "sse", URL: ""}},
		{"unknown_transport_path", &db.MCPServer{Transport: "websocket"}},
	} {
		_, err := connectMCP(context.Background(), c.srv)
		if err == nil {
			t.Fatalf("%s:  ( )", c.label)
		}
		assertKoreanError(t, c.label, err.Error())
	}
}
