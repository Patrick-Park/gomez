package gomez

import (
	"net/textproto"
	"reflect"
	"testing"
)

func TestMessage_AddHeader(t *testing.T) {
	m := NewMessage()

	m.PrependHeader("Names", "Andy")
	m.PrependHeader("Names", "Jane")
	m.PrependHeader("Names", "Bob")

	m.PrependHeader("Types", "Blue")

	if !reflect.DeepEqual(textproto.MIMEHeader{
		"Names": {"Bob", "Jane", "Andy"},
		"Types": {"Blue"},
	}, m.Headers) {
		t.Errorf("Got unexpected result: %#v", m.Headers)
	}
}
