package gomez

import (
	"reflect"
	"testing"
)

func TestAddressCreation(t *testing.T) {
	testCases := []struct {
		testCase string  // Test case
		expct    Address // Expected result address
		err      bool    // Expect error
	}{
		{`John Doe <jerry@seinfeld.com>`, Address{"John Doe", "jerry", "seinfeld.com"}, false},
		{`John jerry@seinfeld.com>`, Address{}, true},
		{`John jerry@seinfeld.com`, Address{}, true},
		{`sosme random" <qwe@asd.com>`, Address{}, true},
		{`"John" jerry@seinfeld.com>`, Address{}, true},
		{`John <jerry@seinfeld.com>`, Address{"John", "jerry", "seinfeld.com"}, false},
		{`"" <>`, Address{}, true},
	}

	for _, test := range testCases {
		addr, err := NewAddress(test.testCase)
		if test.err && err == nil {
			t.Errorf("Expected error on case %s but did not obtain.", test.testCase)
		}

		if !test.err && !reflect.DeepEqual(test.expct, addr) {
			t.Errorf("Expected %#v but got %#v.", test.expct, addr)
		}
	}
}
