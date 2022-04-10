package internal

import (
	"context"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"ethereum-service/proxyClientApi"
	"fmt"
	"os"
)

func GetETHAmount(payment model.Payment) *float64 {
	amount := fmt.Sprintf("%g", payment.PriceAmount)
	srcCurrency := payment.PriceCurrency
	dstCurrency := "ETH" // string |
	mode := "main"

	configuration := NewConfiguration()
	apiClient := proxyClientApi.NewAPIClient(configuration)
	resp, r, err := apiClient.ConversionApi.GetPriceConversion(context.Background()).Amount(amount).SrcCurrency(srcCurrency).DstCurrency(dstCurrency).Mode(mode).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ConversionApi.GetPriceConversion``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	fmt.Fprintf(os.Stdout, "Response from `ConversionApi.GetPriceConversion`: %v\n", resp)
	return resp.Price
}

func NewConfiguration() *proxyClientApi.Configuration {
	cfg := &proxyClientApi.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     "OpenAPI-Generator/1.0.0/go",
		Debug:         true,
		Servers: proxyClientApi.ServerConfigurations{
			{
				URL:         config.Opts.ProxyBaseUrl,
				Description: "No description provided",
			},
		},
		OperationServers: map[string]proxyClientApi.ServerConfigurations{},
	}
	return cfg
}
