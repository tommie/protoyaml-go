package protoyaml

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"gopkg.in/yaml.v3"
)

// Unmarshal interprets the bytes as YAML and populates m.
func Unmarshal(bs []byte, m protoreflect.ProtoMessage) error {
	return NewDecoder(bytes.NewReader(bs)).Decode(m)
}

// A Decoder can be used to decode one or more YAML documents as
// Protobuf messages. It is not goroutine-safe, but is
// goroutine-compatible.
type Decoder struct {
	yd *yaml.Decoder
	r  protoregistry.MessageTypeResolver
}

// NewDecoder creats a new decoder reading from the given stream of
// YAML text.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		yd: yaml.NewDecoder(r),
		r:  protoregistry.GlobalTypes,
	}
}

// MessageTypeResolver sets a custom resolver for anypb.Any types. The
// default in NewDecoder is protoregistry.GlobalTypes.
func (d *Decoder) MessageTypeResolver(r protoregistry.MessageTypeResolver) {
	d.r = r
}

// Decode decodes the next document as a message. The argument can
// either be a proto.Message, or a protoreflect.Message. Returns
// io.EOF if there are no more documents.
func (d *Decoder) Decode(v interface{}) error {
	if v == nil {
		return fmt.Errorf("protoyaml: nil destination message")
	}
	n := &yaml.Node{}
	if err := d.yd.Decode(n); err != nil {
		return err
	}
	if n.Kind == yaml.DocumentNode {
		n = n.Content[0]
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("protoyaml: cannot unmarshal a %v into a %T", n.Kind, v)
	}
	switch m := v.(type) {
	case protoreflect.Message:
		return d.decodeMessage(m, n)
	case protoreflect.ProtoMessage:
		return d.decodeMessage(m.ProtoReflect(), n)
	default:
		return fmt.Errorf("protoyaml: cannot unmarshal into a %T", v)
	}
}

// decodeMessage decodes the given node as a Protobuf message.
func (d *Decoder) decodeMessage(out protoreflect.Message, v *yaml.Node) error {
	if v.Kind == yaml.AliasNode {
		v = v.Alias
	}

	if ok, err := d.decodeKnownType(out, v); err != nil {
		return err
	} else if ok {
		return nil
	}

	if v.Kind != yaml.MappingNode {
		return fmt.Errorf("protoyaml: attempting to decode a %v into a message: %s", v.Kind, out.Descriptor().FullName())
	}

	var key string
	for _, n := range v.Content {
		if key == "" {
			key = n.Value
			continue
		}
		fd := out.Descriptor().Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			return fmt.Errorf("protoyaml: unknown field: %s.%s", out.Descriptor().FullName(), key)
		}
		if err := d.decodeField(out, fd, n); err != nil {
			return err
		}
		key = ""
	}
	return nil
}

// decodeField decodes some value guided by a field descriptor. This
// is the main workhorse of the decoder.
func (d *Decoder) decodeField(out protoreflect.Message, fd protoreflect.FieldDescriptor, v *yaml.Node) error {
	if v.Kind == yaml.AliasNode {
		v = v.Alias
	}

	if fd.IsMap() {
		if v.Kind != yaml.MappingNode {
			return fmt.Errorf("protoyaml: attempting to store a %v in a map field: %s", v.Kind, fd.FullName())
		}

		mp := out.Mutable(fd).Map()
		var key protoreflect.MapKey
		for _, n := range v.Content {
			if key.IsValid() {
				if fd.MapValue().Kind() == protoreflect.MessageKind {
					pv := mp.Mutable(key)
					if err := d.decodeMessage(pv.Message(), n); err != nil {
						return err
					}
				} else {
					pv, err := d.decodeValue(fd.MapValue(), n)
					if err != nil {
						return err
					}
					mp.Set(key, pv)
				}
				key = protoreflect.MapKey{}
			} else {
				pv, err := d.decodeValue(fd.MapKey(), n)
				if err != nil {
					return err
				}
				switch pv.Interface().(type) {
				case bool, int32, int64, uint32, uint64, string:
					// continue
				default:
					return fmt.Errorf("protoyaml: attempting to use %T as a map key in %q", pv.Interface(), fd.FullName())
				}
				key = pv.MapKey()
			}
		}
		return nil
	}

	if fd.IsList() {
		if v.Kind != yaml.SequenceNode {
			return fmt.Errorf("protoyaml: attempting to store a %v in a repeated field: %s", v.Kind, fd.FullName())
		}

		l := out.Mutable(fd).List()
		for _, n := range v.Content {
			if fd.Kind() == protoreflect.MessageKind {
				pv := l.AppendMutable()
				if err := d.decodeMessage(pv.Message(), n); err != nil {
					return err
				}
				continue
			}

			pv, err := d.decodeValue(fd, n)
			if err != nil {
				return err
			}
			l.Append(pv)
		}
		return nil
	}

	if fd.Kind() == protoreflect.MessageKind {
		return d.decodeMessage(out.Mutable(fd).Message(), v)
	}

	switch v.Kind {
	case yaml.ScalarNode:
		pv, err := d.decodeValue(fd, v)
		if err != nil {
			return err
		}

		out.Set(fd, pv)

	default:
		return fmt.Errorf("protoyaml: cannot unmarshal a %v", v.Kind)
	}

	return nil
}

// decodeValue decodes a non-compound value, interpreted based on the
// kind of field it is.
func (d *Decoder) decodeValue(fd protoreflect.FieldDescriptor, v *yaml.Node) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		var vv bool
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(vv), nil

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		var vv int32
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(vv), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		var vv int64
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(vv), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		var vv uint32
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(vv), nil

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		var vv uint64
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(vv), nil

	case protoreflect.FloatKind:
		var vv float32
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(vv), nil

	case protoreflect.DoubleKind:
		var vv float64
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(vv), nil

	case protoreflect.StringKind:
		return protoreflect.ValueOfString(v.Value), nil

	case protoreflect.BytesKind:
		bs, err := base64.StdEncoding.DecodeString(v.Value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBytes(bs), nil

	case protoreflect.EnumKind:
		evd := fd.Enum().Values().ByName(protoreflect.Name(v.Value))
		if evd != nil {
			return protoreflect.ValueOfEnum(evd.Number()), nil
		}

		var vv int32
		if err := v.Decode(&vv); err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(vv)), nil

	default:
		return protoreflect.Value{}, fmt.Errorf("protoyaml: cannot unmarshal a %v into a %v", v.Kind, fd.Kind())
	}
}
