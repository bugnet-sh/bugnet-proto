package gen_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	routerv1 "github.com/oynozan/bugnet-proto/gen/go/bugnet/router/v1"
)

func TestExecRequestRoundTrip(t *testing.T) {
	in := &routerv1.ExecRequest{
		Handle:         "h-1",
		Argv:           []string{"echo", "hi"},
		Cwd:            "/work",
		Env:            map[string]string{"A": "1", "B": "2"},
		IdempotencyKey: "key-1",
		WaitMs:         1500,
		Until: &routerv1.Until{
			Exit:          true,
			OutputPattern: "ready",
			IdleMs:        250,
		},
	}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := &routerv1.ExecRequest{}
	if err := proto.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(in, out) {
		t.Fatalf("round trip mismatch: %v != %v", in, out)
	}
}

func TestWatchJobSkipAndTruncatedListingRoundTrip(t *testing.T) {
	msgs := []proto.Message{
		&routerv1.WatchJobResponse{
			Frame: &routerv1.WatchJobResponse_Skip{Skip: &routerv1.JobSkip{
				StdoutOffset: 1 << 40, StderrOffset: 7,
			}},
			Cursor: "1-0",
		},
		&routerv1.ListFilesResponse{
			Entries:   []*routerv1.FileEntry{{Path: "/a", Size: 3}},
			Truncated: true,
		},
	}
	for _, in := range msgs {
		raw, err := proto.Marshal(in)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		out := in.ProtoReflect().New().Interface()
		if err := proto.Unmarshal(raw, out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !proto.Equal(in, out) {
			t.Fatalf("round trip mismatch: %v != %v", in, out)
		}
	}
}
