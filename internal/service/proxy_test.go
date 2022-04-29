package service

import (
	"ethereum-service/internal/config"
	"ethereum-service/internal/testutils"
	"testing"

	"gopkg.in/h2non/gock.v1"
)

func TestGetETHAmount(t *testing.T) {
	config.ReadOpts()
	expectedPayAmountFloat := 0.0001
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8001").
		Get("/api/price-conversion").
		MatchParam("amount", "100").
		MatchParam("dst_currency", "ETH").
		MatchParam("mode", "main").
		MatchParam("src_currency", "USD").
		Reply(200).
		JSON(map[string]float64{"Price": expectedPayAmountFloat})
	payment := testutils.GetWaitingPayment()
	GetETHAmount(payment)
	if gock.IsDone() != true {
		t.Fatalf("Request should have been sent, but there are open requests")
	}
}
