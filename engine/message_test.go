package engine

import (
	"testing"
)

func TestMarshal(t *testing.T) {
	m := NewMessage("example", "foo", []MessageItem{
		{
			Name:  "Foo",
			Keys:  []string{"id", "title"},
			Types: []string{"int32", "string"},
		},
	})
	data, err := m.Marshal("Foo", []interface{}{int32(1), "hello"})
	if err != nil {
		t.Error(err)
		return
	}
	values, err := m.Unmarshal("Foo", data, []string{"title"})
	if err != nil {
		t.Error(err)
		return
	}
	titleVal, ok := values["title"]
	if !ok {
		t.Error("cant get title value")
		return
	}
	if titleVal.String() != "hello" {
		t.Error("the title value not equal: " + titleVal.String())
		return
	}
}
