package nbp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestShouldRetryNetworkError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"ordinary error", errors.New("invalid payload"), false},
		{"cancellation", context.Canceled, false},
		{"DNS failure", &net.DNSError{Err: "no such host"}, true},
		{"wrapped network failure", fmt.Errorf("request: %w", &net.DNSError{Err: "lookup failed"}), true},
		{"URL failure", &url.Error{Op: "Get", URL: "https://example.invalid", Err: errors.New("connection lost")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryNetworkError(tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

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
