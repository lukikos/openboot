package httputil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_NoRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

func TestDo_RateLimitThenSuccess(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("rate limited"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// Replace sleep to avoid real delays in tests.
	var sleptDuration time.Duration
	originalSleep := sleepFunc
	sleepFunc = func(d time.Duration) { sleptDuration = d }
	defer func() { sleepFunc = originalSleep }()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2*time.Second, sleptDuration)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDo_RateLimitTwiceReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	originalSleep := sleepFunc
	sleepFunc = func(d time.Duration) {}
	defer func() { sleepFunc = originalSleep }()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	assert.Nil(t, resp)
	require.Error(t, err)

	var rateLimitErr *RateLimitError
	require.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 30, rateLimitErr.RetryAfterSeconds)
	assert.Contains(t, rateLimitErr.Error(), "Rate limited")
	assert.Contains(t, rateLimitErr.Error(), "30 seconds")
}

func TestDo_RetryAfterCappedAtMax(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "300") // 5 minutes — should be capped
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var sleptDuration time.Duration
	originalSleep := sleepFunc
	sleepFunc = func(d time.Duration) { sleptDuration = d }
	defer func() { sleepFunc = originalSleep }()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, MaxRetryAfter, sleptDuration)
}

func TestDo_MissingRetryAfterDefaultsToOneSecond(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// No Retry-After header
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var sleptDuration time.Duration
	originalSleep := sleepFunc
	sleepFunc = func(d time.Duration) { sleptDuration = d }
	defer func() { sleepFunc = originalSleep }()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 1*time.Second, sleptDuration)
}

func TestDo_WithRequestBody(t *testing.T) {
	var calls atomic.Int32
	var receivedBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(body))
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalSleep := sleepFunc
	sleepFunc = func(d time.Duration) {}
	defer func() { sleepFunc = originalSleep }()

	payload := []byte(`{"key":"value"}`)
	req, err := http.NewRequest("POST", server.URL, bytes.NewReader(payload))
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), calls.Load())
	// Body should be sent on both attempts.
	require.Len(t, receivedBodies, 2)
	assert.Equal(t, string(payload), receivedBodies[0])
	assert.Equal(t, string(payload), receivedBodies[1])
}

func TestDo_NetworkError(t *testing.T) {
	// Point at a closed server to cause a network error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	req, err := http.NewRequest("GET", serverURL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	assert.Nil(t, resp)
	require.Error(t, err)
}

func TestDo_OtherErrorStatusNotRetried(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := Do(&http.Client{}, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, int32(1), calls.Load()) // Only one call — no retry.
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected int
	}{
		{"valid number", "5", 5},
		{"zero", "0", 0},
		{"negative", "-1", 0},
		{"empty", "", 0},
		{"non-numeric", "abc", 0},
		{"large value", strconv.Itoa(3600), 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}
			assert.Equal(t, tt.expected, parseRetryAfter(resp))
		})
	}
}

func TestClampDuration(t *testing.T) {
	assert.Equal(t, 5*time.Second, clampDuration(5*time.Second, 60*time.Second))
	assert.Equal(t, 60*time.Second, clampDuration(120*time.Second, 60*time.Second))
	assert.Equal(t, 60*time.Second, clampDuration(60*time.Second, 60*time.Second))
}
