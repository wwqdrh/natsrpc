package engine

import (
	"testing"
)

func TestMessageByJson(t *testing.T) {
	jsonData := `
	{
		"syntax": "proto3",
		"packagename": "example",
		"filename": "foo",
		"items": [
			{
				"name": "ReqHello",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "RespHello",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "ReqSend",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "RespSend",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "ReqRequest",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "RespRequest",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "ReqCall",
				"keys": ["name"],
				"types": ["string"]
			},
			{
				"name": "RespCall",
				"keys": ["name"],
				"types": ["string"]
			}
		]
	}
	`
	m := NewMessageByJson([]byte(jsonData))
	data, err := m.Marshal("ReqHello", []interface{}{"hello"})
	if err != nil {
		t.Error(err)
		return
	}
	values, err := m.Unmarshal("ReqHello", data, []string{"name"})
	if err != nil {
		t.Error(err)
		return
	}
	titleVal, ok := values["name"]
	if !ok {
		t.Error("cant get title value")
		return
	}
	if titleVal.String() != "hello" {
		t.Error("the title value not equal: " + titleVal.String())
		return
	}
}

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
