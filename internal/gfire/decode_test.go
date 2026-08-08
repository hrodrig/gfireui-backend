package gfire

import (
	"strings"
	"testing"
)

func TestDecodeQueueSummariesShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		body    string
		wantLen int
		wantErr string
	}{
		{name: "array", body: `[{"name":"default","depth":3}]`, wantLen: 1},
		{name: "queues wrapper", body: `{"queues":[{"name":"a","jobs_count":2}]}`, wantLen: 1},
		{name: "data wrapper", body: `{"data":[{"name":"b","count":1}]}`, wantLen: 1},
		{name: "items wrapper", body: `{"items":[{"name":"c"}]}`, wantLen: 1},
		{name: "result wrapper", body: `{"result":[{"name":"d","depth":9}]}`, wantLen: 1},
		{name: "single object", body: `{"name":"solo","depth":4}`, wantLen: 1},
		{name: "empty", body: `   `, wantErr: "empty"},
		{name: "no queues", body: `{"queues":[]}`, wantErr: "did not contain"},
		{name: "bad json", body: `{`, wantErr: "decode"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeQueueSummaries(strings.NewReader(tt.body))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestDecodeNamedListCountShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		body    string
		keys    []string
		want    int
		wantErr string
	}{
		{name: "array", body: `[{},{}]`, keys: []string{"servers"}, want: 2},
		{name: "named key", body: `{"servers":[{},{},{}]}`, keys: []string{"servers"}, want: 3},
		{name: "fallback key", body: `{"items":[{}]}`, keys: []string{"servers", "items"}, want: 1},
		{name: "total field", body: `{"total":9}`, keys: []string{"servers"}, want: 9},
		{name: "count field", body: `{"count":4}`, keys: []string{"servers"}, want: 4},
		{name: "empty object", body: `{}`, keys: []string{"servers"}, want: 0},
		{name: "empty body", body: `  `, keys: []string{"servers"}, wantErr: "empty"},
		{name: "bad json", body: `{`, keys: []string{"servers"}, wantErr: "decode"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeNamedListCount(strings.NewReader(tt.body), tt.keys...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("count = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDecodeJobCountShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		body    string
		want    int
		wantErr string
	}{
		{name: "array", body: `[{},{}]`, want: 2},
		{name: "jobs wrapper", body: `{"jobs":[{},{},{}]}`, want: 3},
		{name: "data wrapper", body: `{"data":[{}]}`, want: 1},
		{name: "items wrapper", body: `{"items":[{},{}]}`, want: 2},
		{name: "result wrapper", body: `{"result":[{}]}`, want: 1},
		{name: "total field", body: `{"total":7}`, want: 7},
		{name: "count field", body: `{"count":4}`, want: 4},
		{name: "jobs_count field", body: `{"jobs_count":5}`, want: 5},
		{name: "empty", body: ``, wantErr: "empty"},
		{name: "bad json", body: `{`, wantErr: "decode"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeJobCount(strings.NewReader(tt.body))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("count = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestJoinURLPathAndLeadingSlash(t *testing.T) {
	t.Parallel()

	if got := joinURLPath("", ""); got != "/" {
		t.Fatalf("join empty = %q", got)
	}
	if got := joinURLPath("", "v1"); got != "/v1" {
		t.Fatalf("join relative = %q", got)
	}
	if got := joinURLPath("/base", ""); got != "/base" {
		t.Fatalf("join base only = %q", got)
	}
	if got := joinURLPath("/base/", "/v1/jobs"); got != "/base/v1/jobs" {
		t.Fatalf("join both = %q", got)
	}
	if got := ensureLeadingSlash("x"); got != "/x" {
		t.Fatalf("ensure = %q", got)
	}
	if got := ensureLeadingSlash("/x"); got != "/x" {
		t.Fatalf("ensure kept = %q", got)
	}
}

func TestNewClientRejectsEmptyBaseURL(t *testing.T) {
	t.Parallel()
	if _, err := NewClient("", "tok", nil); err == nil {
		t.Fatal("expected error")
	}
	if _, err := NewClient("://bad", "tok", nil); err == nil {
		t.Fatal("expected parse error")
	}
}
