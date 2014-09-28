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
