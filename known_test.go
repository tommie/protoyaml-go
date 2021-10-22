package protoyaml

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/tommie/protoyaml-go/internal/testproto"
)

func TestDecoderDecodeAny(t *testing.T) {
	d, n, err := parseYAML(`{"@type": "type.googleapis.com/protoyaml.test.Message", astring: "hello"}`)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}
	var got anypb.Any
	if err := d.decodeAny(got.ProtoReflect(), n); err != nil {
		t.Fatalf("decodeAny failed: %v", err)
	}

	want, err := anypb.New(&testproto.Message{Astring: "hello"})
	if err != nil {
		t.Fatalf("anypb.New failed: %v", err)
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("decodeAny: +got, -want:\n%s", diff)
	}
}

func TestDecoderDecodeDuration(t *testing.T) {
	d, n, err := parseYAML(`"42s"`)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}
	var got durationpb.Duration
	if err := d.decodeDuration(got.ProtoReflect(), n); err != nil {
		t.Fatalf("decodeDuration failed: %v", err)
	}

	want := durationpb.New(42 * time.Second)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("decodeDuration: +got, -want:\n%s", diff)
	}
}

func TestDecoderDecodeFieldMask(t *testing.T) {
	d, n, err := parseYAML(`["amessage.anint32", "astring"]`)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}
	var got fieldmaskpb.FieldMask
	if err := d.decodeFieldMask(got.ProtoReflect(), n); err != nil {
		t.Fatalf("decodeFieldMask failed: %v", err)
	}

	want, err := fieldmaskpb.New(&testproto.Message{}, "amessage.anint32", "astring")
	if err != nil {
		t.Fatalf("fieldmaskpb.New failed: %v", err)
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("decodeFieldMask: +got, -want:\n%s", diff)
	}
}

func TestDecoderDecodeTimestamp(t *testing.T) {
	d, n, err := parseYAML(`"2006-01-02T15:04:05.999Z"`)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}
	var got timestamppb.Timestamp
	if err := d.decodeTimestamp(got.ProtoReflect(), n); err != nil {
		t.Fatalf("decodeTimestamp failed: %v", err)
	}

	want := timestamppb.New(time.Date(2006, 1, 2, 15, 4, 5, 999000000, time.UTC))
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("decodeTimestamp: +got, -want:\n%s", diff)
	}
}
