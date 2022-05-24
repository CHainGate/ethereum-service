package config

import (
	"os"
	"testing"
)

func setup() {
	ReadOpts()
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
	CreateTestClientConnection(Opts.Test)
	if ClientTest == nil {
		t.Fatalf(`clientTest should not be nil`)
	}
}

func TestConnectingToMainnet(t *testing.T) {
	CreateMainClientConnection(Opts.Main)
	if ClientMain == nil {
		t.Fatalf(`clientMain should not be nil`)
	}
}
