package traefik_real_ip_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	plugin "github.com/xethlyx/traefik-real-ip"
)

func TestNew(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.TrustedIPs = []string{"10.0.0.0/24"}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		desc                 string
		remoteAddr           string
		header               string
		value                string
		expectedRealIp       string
		expectedForwardedFor []string
	}{
		{
			desc:                 "don't forward",
			remoteAddr:           "10.0.1.0:9000",
			header:               "X-Forwarded-For",
			value:                "127.0.0.2, 10.0.1.0",
			expectedRealIp:       "10.0.1.0",
			expectedForwardedFor: []string{"10.0.1.0"},
		},
		{
			desc:                 "don't forward multiple",
			remoteAddr:           "10.0.1.0:9000",
			header:               "X-Forwarded-For",
			value:                "127.0.0.2, 10.0.0.1, 10.0.1.0",
			expectedRealIp:       "10.0.1.0",
			expectedForwardedFor: []string{"10.0.1.0"},
		},
		{
			desc:                 "overwrite real ip",
			remoteAddr:           "10.0.1.0:9000",
			header:               "X-Real-Ip",
			value:                "127.0.0.2, 10.0.1.0",
			expectedRealIp:       "10.0.1.0",
			expectedForwardedFor: []string{"10.0.1.0"},
		},
		{
			desc:                 "forward",
			remoteAddr:           "10.0.0.1:9000",
			header:               "X-Forwarded-For",
			value:                "1.1.1.1, 10.0.0.1",
			expectedRealIp:       "1.1.1.1",
			expectedForwardedFor: []string{"1.1.1.1", "10.0.0.1"},
		},
		{
			desc:                 "forward multiple",
			remoteAddr:           "10.0.0.1:9000",
			header:               "X-Forwarded-For",
			value:                "10.0.0.3, 1.1.1.1, 10.0.0.20, 10.0.0.1",
			expectedRealIp:       "1.1.1.1",
			expectedForwardedFor: []string{"1.1.1.1", "10.0.0.20", "10.0.0.1"},
		},
		{
			desc:                 "forward empty",
			remoteAddr:           "10.0.0.1:9000",
			header:               "X-Forwarded-For",
			value:                "",
			expectedRealIp:       "10.0.0.1",
			expectedForwardedFor: []string{"10.0.0.1"},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = test.remoteAddr

			req.Header.Set(test.header, test.value)

			handler.ServeHTTP(recorder, req)

			if req.Header.Get("X-Real-Ip") != test.expectedRealIp {
				t.Errorf("invalid X-Real-Ip value: %s", req.Header.Get("X-Real-Ip"))
			}

			if req.Header.Get("X-Forwarded-For") != strings.Join(test.expectedForwardedFor, ", ") {
				t.Errorf("invalid X-Forwarded-For value: %s", req.Header.Get("X-Forwarded-For"))
			}
		})
	}
}
