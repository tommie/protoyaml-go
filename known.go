package protoyaml

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v3"
)

func (d *Decoder) decodeKnownType(out protoreflect.Message, v *yaml.Node) (bool, error) {
	switch out.Type() {
	case durationType:
		return true, d.decodeDuration(out, v)
	default:
		return false, nil
	}

	return true, nil
}

var (
	durationType = (&durationpb.Duration{}).ProtoReflect().Type()
)

func (d *Decoder) decodeDuration(out protoreflect.Message, v *yaml.Node) error {
	dur := out.Interface().(*durationpb.Duration)

	if v.Kind != yaml.ScalarNode {
		return fmt.Errorf("protoyaml: attempting to unmarshal a %v into a durationpb.Duration", v.Kind)
	}

	return protojson.Unmarshal([]byte(strconv.Quote(v.Value)), dur)
}
