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
	m.Headers.Prepend("Types", "Green")
	m.Headers.Add("Types", "Red")

	if !reflect.DeepEqual(OrderedHeader{textproto.MIMEHeader{
		"Names": {"Bob", "Jane", "Andy"},
		"Types": {"Green", "Blue", "Red"},
	}}, m.Headers) {
		t.Errorf("Got unexpected result: %#v", m.Headers)
	}
}

func TestMessage_FromRaw(t *testing.T) {
	testSuite := []struct {
		raw         string
		msg         Message
		isCompliant bool
	}{
		{`From: Some guy
Date: 23 Jan 2014
Subject: Hi

Hey what's up?`, Message{
			Headers: OrderedHeader{textproto.MIMEHeader{
				"From":    []string{"Some guy"},
				"Date":    []string{"23 Jan 2014"},
				"Subject": []string{"Hi"},
			}},
			Body: "Hey what's up?",
		}, true},
		{`Date: 1 Dec 2018
From: Jim

Hello
Was gonna ask about you`, Message{
			Headers: OrderedHeader{textproto.MIMEHeader{
				"From": []string{"Jim"},
				"Date": []string{"1 Dec 2018"},
			}},
			Body: "Hello\r\nWas gonna ask about you",
		}, true},
		{`From: Some guy
Date: 23 Jan 2014
Subject: Hi
`, Message{}, false},
		{`From: Some guy
Subject: Hi

Hey what's up?`, Message{}, false},
		{`Hey what's up?`, Message{}, false},
		{``, Message{}, false},
	}

	for _, test := range testSuite {
		m := NewMessage()
		err := m.FromRaw(test.raw)

		if test.isCompliant && (!reflect.DeepEqual(m.Headers, test.msg.Headers) || test.msg.Body != m.Body) {
			t.Errorf("Expected %#v and %s, but got \r\n\r\n%#v and %s", test.msg.Headers, test.msg.Body, m.Headers, m.Body)
		}

		if !test.isCompliant && err == nil {
			t.Errorf("Incompliant message passed as compliant: %s", test.raw)
		}
	}
}

func TestMessage_Raw_And_Back(t *testing.T) {
	testSuite := []Message{
		Message{
			Headers: OrderedHeader{textproto.MIMEHeader{
				"From": []string{"Jim"},
				"Date": []string{"1 Jan 1989"},
				"To":   []string{"Gophers"},
			}},
			Body: "Hey Gophers!\r\nHow's it going these days?"},

		Message{
			Headers: OrderedHeader{textproto.MIMEHeader{
				"Subject":  []string{"Hello"},
				"Date":     []string{"1 Jan 1989"},
				"From":     []string{"Gophers"},
				"Received": []string{"John at 123", "Jim at 250"},
			}},
			Body: "Hey Foo!\r\nHow's it going these days?"},
	}

	for _, test := range testSuite {
		m := NewMessage()

		m.FromRaw(test.Raw())
		m.FromRaw(m.Raw())

		if !reflect.DeepEqual(*m, test) {
			t.Errorf("Expected: \r\n%#v, got:\r\n\r\n%#v\r\n\r\n", test, m)
		}
	}
}
