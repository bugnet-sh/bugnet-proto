package gen_test

import (
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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

// wire is a field's number and its type as a .proto file declares it
type wire struct {
	num protoreflect.FieldNumber
	typ string
}

// checkWire fails for each field of pkg whose number or declared type differs from want
func checkWire(t *testing.T, pkg string, want map[string]wire) {
	t.Helper()
	for name, w := range want {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(
			protoreflect.FullName(pkg + "." + name))
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		fd := d.(protoreflect.FieldDescriptor)
		if got := protoType(fd); fd.Number() != w.num || got != w.typ {
			t.Errorf("%s is %s = %d, want %s = %d", name, got, fd.Number(), w.typ, w.num)
		}
	}
}

// protoType spells a field's type as its .proto file declares it, with the file's package dropped
func protoType(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return "map<" + protoType(fd.MapKey()) + ", " + protoType(fd.MapValue()) + ">"
	}
	name := fd.Kind().String()
	var full protoreflect.FullName
	if m := fd.Message(); m != nil {
		full = m.FullName()
	} else if e := fd.Enum(); e != nil {
		full = e.FullName()
	}
	if full != "" {
		name = strings.TrimPrefix(string(full), string(fd.ParentFile().Package())+".")
	}
	switch {
	case fd.Cardinality() == protoreflect.Repeated:
		return "repeated " + name
	case fd.HasOptionalKeyword():
		return "optional " + name
	}
	return name
}
