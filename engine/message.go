package engine

import (
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// now support
// - int32
// - string
// - bool

type message struct {
	Syntax      string        `json:"syntax"`
	PackageName string        `json:"package"`
	FileName    string        `json:"file"`
	Items       []MessageItem `json:"items"`

	fd pref.FileDescriptor `json:"-"`
}

type MessageItem struct {
	Name     string          `json:"name"`
	Keys     []string        `json:"keys"`  // name, example id, title, content
	Types    []string        `json:"types"` // int string bool
	Repeated map[string]bool `json:"repeated"`
	Optional map[string]bool `json:"optional"`
}

func NewMessage(packageName string, filename string, fields []MessageItem) *message {
	return &message{
		Syntax:      "proto3",
		PackageName: packageName,
		FileName:    filename,
		Items:       fields[:],
	}
}

func NewMessageByJson(data []byte) *message {
	var res message
	json.Unmarshal(data, &res)
	return &res
}

func (m *MessageItem) GetDescriptor() *descriptorpb.DescriptorProto {
	dp := &descriptorpb.DescriptorProto{
		Name:  proto.String(m.Name),
		Field: []*descriptorpb.FieldDescriptorProto{},
	}
	keys := m.Keys
	types := m.Types
	for i, key := range keys {
		var (
			curType     *descriptorpb.FieldDescriptorProto_Type
			curLabel    *descriptorpb.FieldDescriptorProto_Label
			curTypeName *string
		)

		switch types[i] {
		case "int32":
			curType = descriptorpb.FieldDescriptorProto_Type(pref.Int32Kind).Enum()
		case "string":
			curType = descriptorpb.FieldDescriptorProto_Type(pref.StringKind).Enum()
		case "bool":
			curType = descriptorpb.FieldDescriptorProto_Type(pref.BoolKind).Enum()
		default:
			curType = descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum()
			curTypeName = proto.String(types[i])
		}

		if m.Repeated != nil && m.Repeated[key] {
			curLabel = descriptorpb.FieldDescriptorProto_Label(pref.Repeated).Enum()
		}

		if m.Optional != nil && m.Optional[key] {
			curLabel = descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum()
		}

		dp.Field = append(dp.Field, &descriptorpb.FieldDescriptorProto{
			Name:     proto.String(key),
			JsonName: proto.String(key),
			Number:   proto.Int32(int32(i) + 1),
			Label:    curLabel,
			Type:     curType,
			TypeName: curTypeName,
		})
	}

	return dp
}

func (m *message) init() error {
	pb := &descriptorpb.FileDescriptorProto{
		Syntax:      proto.String(m.Syntax),
		Name:        proto.String(fmt.Sprintf("%s.%s", m.PackageName, m.FileName)),
		Package:     proto.String(m.PackageName),
		MessageType: []*descriptorpb.DescriptorProto{},
	}
	for _, item := range m.Items {
		pb.MessageType = append(pb.MessageType, item.GetDescriptor())
	}

	fd, err := protodesc.NewFile(pb, nil)
	if err != nil {
		return err
	}
	m.fd = fd
	return nil
}

func (m *message) Marshal(name string, values []interface{}) ([]byte, error) {
	if m.fd == nil {
		if err := m.init(); err != nil {
			return nil, err
		}
	}
	var messageItem MessageItem
	var messageItemFound bool
	for _, item := range m.Items {
		if item.Name == name {
			messageItem = item
			messageItemFound = true
			break
		}
	}
	if !messageItemFound {
		return nil, errors.New(name + " message not found")
	}

	fooMessageDescriptor := m.fd.Messages().ByName(pref.Name(name))
	msg := dynamicpb.NewMessage(fooMessageDescriptor)
	keys := messageItem.Keys
	types := messageItem.Types
	for i, key := range keys {
		switch types[i] {
		case "int32":
			value, ok := values[i].(int32)
			if !ok {
				return nil, errors.New("field not int32 type")
			}
			msg.Set(fooMessageDescriptor.Fields().ByName(pref.Name(key)), pref.ValueOfInt32(value))
		case "string":
			value, ok := values[i].(string)
			if !ok {
				return nil, errors.New("field not int32 type")
			}
			msg.Set(fooMessageDescriptor.Fields().ByName(pref.Name(key)), pref.ValueOfString(value))
		case "bool":
			value, ok := values[i].(bool)
			if !ok {
				return nil, errors.New("field not int32 type")
			}
			msg.Set(fooMessageDescriptor.Fields().ByName(pref.Name(key)), pref.ValueOfBool(value))
		}
	}

	if data, err := proto.Marshal(msg); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (m *message) Unmarshal(name string, data []byte, fields []string) (map[string]pref.Value, error) {
	if m.fd == nil {
		if err := m.init(); err != nil {
			return nil, err
		}
	}

	barMessageDescriptor := m.fd.Messages().ByName(pref.Name(name))
	msg := dynamicpb.NewMessage(barMessageDescriptor)
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, err
	}

	result := map[string]pref.Value{}
	for _, field := range fields {
		result[field] = msg.Get(barMessageDescriptor.Fields().ByName(pref.Name(field)))
	}
	return result, nil
}
