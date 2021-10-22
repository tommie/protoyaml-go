package protoyaml

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

func (d *Decoder) decodeKnownType(out protoreflect.Message, v *yaml.Node) (bool, error) {
	switch out.Type() {
	case anyType:
		return true, d.decodeAny(out, v)
	case durationType:
		return true, d.decodeDuration(out, v)
	case timestampType:
		return true, d.decodeTimestamp(out, v)
	default:
		return false, nil
	}

	return true, nil
}

var (
	anyType       = (&anypb.Any{}).ProtoReflect().Type()
	durationType  = (&durationpb.Duration{}).ProtoReflect().Type()
	timestampType = (&timestamppb.Timestamp{}).ProtoReflect().Type()
)

func (d *Decoder) decodeAny(out protoreflect.Message, v *yaml.Node) error {
	any := out.Interface().(*anypb.Any)

	if v.Kind != yaml.MappingNode {
		return fmt.Errorf("protoyaml: attempting to unmarshal a %v into an anypb.Any", v.Kind)
	}

	var mt protoreflect.MessageType
	var typeIndex int
	var key string
	for i, n := range v.Content {
		if key == "" {
			key = n.Value
			continue
		}
		switch key {
		case "@type":
			var err error
			mt, err = d.r.FindMessageByURL(n.Value)
			if err != nil {
				return err
			}
			any.TypeUrl = n.Value
			typeIndex = i - 1
		}
		key = ""
	}

	if mt == nil {
		return fmt.Errorf("protoyaml: no @type key in Any mapping")
	}

	n := *v
	n.Content = append(append([]*yaml.Node{}, v.Content[:typeIndex]...), v.Content[typeIndex+2:]...)

	m := mt.New()
	if err := d.decodeMessage(m, &n); err != nil {
		return err
	}

	return any.MarshalFrom(m.Interface())
}

func (d *Decoder) decodeDuration(out protoreflect.Message, v *yaml.Node) error {
	dur := out.Interface().(*durationpb.Duration)

	if v.Kind != yaml.ScalarNode {
		return fmt.Errorf("protoyaml: attempting to unmarshal a %v into a durationpb.Duration", v.Kind)
	}

	return protojson.Unmarshal([]byte(strconv.Quote(v.Value)), dur)
}

func (d *Decoder) decodeTimestamp(out protoreflect.Message, v *yaml.Node) error {
	dur := out.Interface().(*timestamppb.Timestamp)

	if v.Kind != yaml.ScalarNode {
		return fmt.Errorf("protoyaml: attempting to unmarshal a %v into a timestamppb.Timestamp", v.Kind)
	}

	return protojson.Unmarshal([]byte(strconv.Quote(v.Value)), dur)
}
