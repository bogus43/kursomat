package nbp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestMapNetworkError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		err, want error
	}{
		{"deadline", context.DeadlineExceeded, ErrTimeout},
		{"wrapped deadline", fmt.Errorf("request: %w", context.DeadlineExceeded), ErrTimeout},
		{"URL deadline", &url.Error{Op: "Get", URL: "https://example.invalid", Err: context.DeadlineExceeded}, ErrTimeout},
		{"DNS timeout", &net.DNSError{Err: "timeout", IsTimeout: true}, ErrTimeout},
		{"wrapped DNS timeout", fmt.Errorf("request: %w", &net.DNSError{Err: "timeout", IsTimeout: true}), ErrTimeout},
		{"DNS failure", &net.DNSError{Err: "no such host", IsNotFound: true}, ErrConnection},
		{"other failure", errors.New("connection lost"), ErrConnection},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapNetworkError(tc.err); !errors.Is(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
