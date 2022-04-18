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
