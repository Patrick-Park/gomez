package gomez

import (
	"net/textproto"
	"reflect"
	"testing"
)

func TestMessage_AddHeader(t *testing.T) {
	m := new(Message)

	m.AddHeader("Names", "Andy")
	m.AddHeader("Names", "Jane")
	m.AddHeader("Names", "Bob")

	m.AddHeader("Types", "Blue")

	if !reflect.DeepEqual(textproto.MIMEHeader{
		"Names": {"Bob", "Jane", "Andy"},
		"Types": {"Blue"},
	}, m.Headers) {
		t.Errorf("Got unexpected result: %#v", m.Headers)
	}
}
