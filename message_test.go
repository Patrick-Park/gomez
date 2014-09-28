package gomez

import (
	"net/mail"
	"reflect"
	"testing"
)

func TestMessage_Header(t *testing.T) {
	testSuite := []struct {
		Message string
		Header  mail.Header
		HasErr  bool
	}{
		{`Subject: Hello!
From: Jim`, mail.Header{}, true},

		{``, mail.Header{}, true},

		{`just some random text`, mail.Header{}, true},

		{`Date: Today
Received: ABCD
Subject: Hello
Received: QWER

Hey how are you?`, mail.Header{
			"Subject":  []string{"Hello"},
			"Received": []string{"ABCD", "QWER"},
			"Date":     []string{"Today"},
		}, false},

		{`Random-Header: My value

`, mail.Header{
			"Random-Header": []string{"My value"},
		}, false},
	}

	for _, test := range testSuite {
		m := Message{Raw: test.Message}
		h, err := m.Header()

		if test.HasErr && err == nil {
			t.Errorf("Was expecting error on %s", test.Message)
		} else if !test.HasErr && !reflect.DeepEqual(h, test.Header) {
			t.Errorf("Did not parse headers correctly: expected %+v, got: %+v", test.Header, h)
		}
	}
}
