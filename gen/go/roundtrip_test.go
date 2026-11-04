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
