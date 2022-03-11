package main

import "testing"

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestTest(t *testing.T) {
	value := getTest()
	if !(value == "Test:") {
		t.Fatalf(`%q, want match for %q, nil`, value, "Test:")
	}
}
