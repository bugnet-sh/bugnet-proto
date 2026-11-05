package gen_test

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
)

// eq fails the test when got and want differ
func eq[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// roundTrip marshals in and returns what unmarshals from the bytes
func roundTrip[M proto.Message](t *testing.T, in M) M {
	t.Helper()
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := in.ProtoReflect().New().Interface().(M)
	if err := proto.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
