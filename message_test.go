package gomez

import (
	"net/textproto"
	"reflect"
	"testing"
)

func TestMessage_AddHeader(t *testing.T) {
	m := NewMessage()

	m.Headers.Prepend("Names", "Andy")
	m.Headers.Prepend("Names", "Jane")
	m.Headers.Prepend("Names", "Bob")

	m.Headers.Set("Types", "Blue")

	if !reflect.DeepEqual(OrderedHeader{textproto.MIMEHeader{
		"Names": {"Bob", "Jane", "Andy"},
		"Types": {"Blue"},
	}}, m.Headers) {
		t.Errorf("Got unexpected result: %#v", m.Headers)
	}
}
