package gomez

import (
	"io/ioutil"
	"net/mail"
	"reflect"
	"testing"
)

func TestMessage_Parse(t *testing.T) {
	testSuite := []struct {
		Message string
		Header  mail.Header
		Body    string
		HasErr  bool
	}{
		{`Subject: Hello!
From: Jim`, mail.Header{}, "", true},

		{``, mail.Header{}, "", true},

		{`just some random text`, mail.Header{}, "", true},

		{`Date: Today
Received: ABCD
Subject: Hello
Received: QWER

Hey how are you?`, mail.Header{
			"Subject":  []string{"Hello"},
			"Received": []string{"ABCD", "QWER"},
			"Date":     []string{"Today"},
		}, "Hey how are you?", false},

		{`Random-Header: My value

`, mail.Header{
			"Random-Header": []string{"My value"},
		}, "", false},

		{`Random-Header: My value

This is
a message
on more than
one line.`, mail.Header{
			"Random-Header": []string{"My value"},
		}, "This is\na message\non more than\none line.", false},
	}

	for _, test := range testSuite {
		m := Message{Raw: test.Message}
		msg, err := m.Parse()

		if test.HasErr && err == nil {
			t.Errorf("Was expecting error on %s", test.Message)
		} else if !test.HasErr {
			body, err := ioutil.ReadAll(msg.Body)
			if err != nil {
				t.Errorf("Failed to read body: %+v", err)
			}

			if !reflect.DeepEqual(test.Header, msg.Header) || string(body) != test.Body {
				t.Errorf("Did not parse headers correctly: expected %+v, got: %+v\r\nAnd body: %s, but got %s", test.Header, msg.Header, test.Body, string(body))
			}
		}
	}
}

func TestMessage_Setters_Getters(t *testing.T) {
	m := new(Message)

	addrList := []*mail.Address{
		&mail.Address{"James", "james@harding.com"},
		&mail.Address{"Jones", "jones@google.com"},
		&mail.Address{"Mike", "mike@harding.com"},
	}

	for _, address := range addrList {
		m.AddRcpt(address)
		m.SetFrom(address)
		if m.From() != address {
			t.Error("Did not set FROM correctly")
		}
	}

	if !reflect.DeepEqual(m.Rcpt(), addrList) {
		t.Error("Did not add/retrieve recipients correctly.")
	}
}

func TestMessage_PrependHeader(t *testing.T) {
	testSuite := []struct {
		Message    string
		Key, Value string
		Expected   string
	}{
		{
			"", "Key", "Value",
			"Key: Value\r\n",
		},
		{
			"There is some text", "Key", "Value",
			"Key: Value\r\nThere is some text",
		},
		{
			"There is some text", "Key", "Value on\r\n\tmultiple lines",
			"Key: Value on\r\n\tmultiple lines\r\nThere is some text",
		},
	}

	for _, test := range testSuite {
		m := &Message{Raw: test.Message}
		m.PrependHeader(test.Key, test.Value)

		if m.Raw != test.Expected {
			t.Errorf("Header not prepended correctly. Was expecting:\r\n\r\n%s\r\n\r\nbut got:\r\n\r\n%s",
				test.Expected,
				m.Raw)
		}
	}
}
