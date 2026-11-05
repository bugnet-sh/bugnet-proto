package gen_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func TestCheckExtendPlacementRoundTrip(t *testing.T) {
	deadline := timestamppb.New(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	msgs := []proto.Message{
		&routerv1.CheckExtendPlacementRequest{Handle: "h-1", PackageId: "small-10m-more"},
		&routerv1.CheckExtendPlacementResponse{
			Placement: &routerv1.Placement{
				PlacementId: "p-1", State: routerv1.PlacementState_PLACEMENT_STATE_READY,
			},
			Deadline: deadline,
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

func TestExtensionPricingRoundTrip(t *testing.T) {
	more := &routerv1.PackageOption{
		PackageId: "small-10m-more", Resources: &routerv1.Resources{Vcpu: 2, MemoryMib: 2048},
		Minutes: 10, PriceMicros: 45000, Available: true,
	}
	msgs := []proto.Message{
		&routerv1.CheckExtendPlacementResponse{
			Placement: &routerv1.Placement{PlacementId: "p-1"}, Package: more,
		},
		&routerv1.ListExtensionsRequest{Handle: "h-1"},
		&routerv1.ListExtensionsResponse{Options: []*routerv1.PackageOption{
			more, {PackageId: "small-10m", Minutes: 10, UnavailableReason: "exceeds_lifetime"},
		}},
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
