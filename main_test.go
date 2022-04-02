package main

import (
	"ethereum-service/internal/config"
	"os"
	"testing"
)

func setup() {
	config.ReadOpts()
}

func shutdown() {
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestTest(t *testing.T) {
	value := getTest()
	if !(value == "Test:") {
		t.Fatalf(`%q, want match for %q, nil`, value, "Test:")
	}
}

func TestConnectingToTestnet(t *testing.T) {
	createTestClientConnection(config.Opts.Test)
	if clientTest == nil {
		t.Fatalf(`clientTest should not be nil`)
	}
}

func TestConnectingToMainnet(t *testing.T) {
	createMainClientConnection(config.Opts.Main)
	if clientMain == nil {
		t.Fatalf(`clientMain should not be nil`)
	}
}
