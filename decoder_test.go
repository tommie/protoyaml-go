package protoyaml

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v3"

	"github.com/tommie/protoyaml-go/internal/testproto"
)

func ExampleUnmarshal() (*testproto.Message, error) {
	var got testproto.Message
	if err := Unmarshal([]byte(`astring: hello`), &got); err != nil {
		return nil, err
	}
	return &got, nil
}

func ExampleDecoder_Decode(yamlString string) ([]*testproto.Message, error) {
	d := NewDecoder(strings.NewReader(yamlString))

	var outs []*testproto.Message
	for {
		var m testproto.Message
		err := d.Decode(&m)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		outs = append(outs, &m)
	}

	return outs, nil
}

func TestUnmarshal(t *testing.T) {
	var got testproto.Message
	if err := Unmarshal([]byte(`astring: hello`), &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	want := testproto.Message{Astring: "hello"}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("Unmarshal: +got, -want:\n%s", diff)
	}
}

func TestDecoderDecode(t *testing.T) {
	t.Run("Message", func(t *testing.T) {
		var got testproto.Message
		if err := NewDecoder(strings.NewReader(`astring: hello`)).Decode(got.ProtoReflect()); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		want := testproto.Message{Astring: "hello"}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Decode: +got, -want:\n%s", diff)
		}
	})

	t.Run("multipleDocuments", func(t *testing.T) {
		d := NewDecoder(strings.NewReader(`astring: hello
---
anint32: 42`))

		var got []*testproto.Message
		for {
			var m testproto.Message
			if err := d.Decode(&m); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			got = append(got, &m)
		}

		want := []*testproto.Message{{Astring: "hello"}, {Anint32: 42}}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Decode: +got, -want:\n%s", diff)
		}
	})
}

func TestDecoderDecodeMessage(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		d, n, err := parseYAML(`anint32: 42
astring: hello`)
		if err != nil {
			t.Fatalf("parseYAML failed: %v", err)
		}
		var got testproto.Message
		if err := d.decodeMessage(got.ProtoReflect(), n); err != nil {
			t.Fatalf("decodeMessage failed: %v", err)
		}

		want := testproto.Message{Anint32: 42, Astring: "hello"}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("decodeMessage: +got, -want:\n%s", diff)
		}
	})

	t.Run("known", func(t *testing.T) {
		d, n, err := parseYAML(`aduration: "42s"`)
		if err != nil {
			t.Fatalf("parseYAML failed: %v", err)
		}
		var got testproto.Known
		if err := d.decodeMessage(got.ProtoReflect(), n); err != nil {
			t.Fatalf("decodeMessage failed: %v", err)
		}

		want := testproto.Known{Aduration: durationpb.New(42 * time.Second)}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("decodeMessage: +got, -want:\n%s", diff)
		}
	})
}

func TestDecoderDecodeField(t *testing.T) {
	fds := (&testproto.Message{}).ProtoReflect().Descriptor().Fields()
	tsts := []struct {
		Name string
		YAML string
		FD   protoreflect.FieldDescriptor
		Want testproto.Message
	}{
		{"scalar", `42`, fds.ByName("anint32"), testproto.Message{Anint32: 42}},

		{"scalarSequence", `[42, 43]`, fds.ByName("arepeated_int32"), testproto.Message{ArepeatedInt32: []int32{42, 43}}},
		{"messageSequence", `[{anint64: 42}, {anint64: 43}]`, fds.ByName("arepeated_message"), testproto.Message{ArepeatedMessage: []*testproto.Message{{Anint64: 42}, {Anint64: 43}}}},
		{"scalarSequence", `[42, 43]`, fds.ByName("arepeated_int32"), testproto.Message{ArepeatedInt32: []int32{42, 43}}},

		{"messageMapping", `{anint32: 42}`, fds.ByName("amessage"), testproto.Message{Amessage: &testproto.Message{Anint32: 42}}},
		{"scalarMapMapping", `{anykey: 42}`, fds.ByName("astring_int32_map"), testproto.Message{AstringInt32Map: map[string]int32{"anykey": 42}}},
		{"messageMapMapping", `{anykey: {anint32: 42}}`, fds.ByName("astring_message_map"), testproto.Message{AstringMessageMap: map[string]*testproto.Message{"anykey": &testproto.Message{Anint32: 42}}}},
	}
	for _, tst := range tsts {
		t.Run(tst.Name, func(t *testing.T) {
			d, n, err := parseYAML(tst.YAML)
			if err != nil {
				t.Fatalf("parseYAML failed: %v", err)
			}
			var got testproto.Message
			if err := d.decodeField(got.ProtoReflect(), tst.FD, n); err != nil {
				t.Fatalf("decodeField failed: %v", err)
			}

			if diff := cmp.Diff(tst.Want, got, protocmp.Transform()); diff != "" {
				t.Errorf("decodeField: +got, -want:\n%s", diff)
			}
		})
	}
}

func TestDecoderDecodeValue(t *testing.T) {
	fds := (&testproto.Message{}).ProtoReflect().Descriptor().Fields()
	tsts := []struct {
		Name string
		YAML string
		FD   protoreflect.FieldDescriptor
		Want interface{}
	}{
		{"false", `false`, fds.ByName("abool"), false},
		{"true", `true`, fds.ByName("abool"), true},

		{"int32", `42`, fds.ByName("anint32"), int32(42)},
		{"sint32", `42`, fds.ByName("ansint32"), int32(42)},
		{"sfixed32", `42`, fds.ByName("ansfixed32"), int32(42)},

		{"int64", `42`, fds.ByName("anint64"), int64(42)},
		{"sint64", `42`, fds.ByName("ansint64"), int64(42)},
		{"sfixed64", `42`, fds.ByName("ansfixed64"), int64(42)},

		{"uint32", `42`, fds.ByName("auint32"), uint32(42)},
		{"fixed32", `42`, fds.ByName("afixed32"), uint32(42)},

		{"uint64", `42`, fds.ByName("auint64"), uint64(42)},
		{"fixed64", `42`, fds.ByName("afixed64"), uint64(42)},

		{"float", `42.5`, fds.ByName("afloat"), float32(42.5)},
		{"double", `42.5`, fds.ByName("adouble"), float64(42.5)},

		{"string", `"hello world"`, fds.ByName("astring"), "hello world"},
		{"bytes", `"AAAA"`, fds.ByName("abytes"), []byte{0, 0, 0}},

		{"enumName", `ONE`, fds.ByName("anenum"), testproto.Enum_ONE.Number()},
		{"enumNumber", `1`, fds.ByName("anenum"), testproto.Enum_ONE.Number()},
	}
	for _, tst := range tsts {
		t.Run(tst.Name, func(t *testing.T) {
			d, n, err := parseYAML(tst.YAML)
			if err != nil {
				t.Fatalf("parseYAML failed: %v", err)
			}
			got, err := d.decodeValue(tst.FD, n)
			if err != nil {
				t.Fatalf("decodeValue failed: %v", err)
			}

			if diff := cmp.Diff(tst.Want, got.Interface(), protocmp.Transform()); diff != "" {
				t.Errorf("decodeValue: +got, -want:\n%s", diff)
			}
		})
	}
}

func parseYAML(s string) (*Decoder, *yaml.Node, error) {
	d := NewDecoder(strings.NewReader(s))
	var n yaml.Node
	if err := d.yd.Decode(&n); err != nil {
		return nil, nil, err
	}
	return d, n.Content[0], nil
}
