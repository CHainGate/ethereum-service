package service

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

	configuration := proxyClientApi.NewConfiguration()
	configuration.Servers[0].URL = config.Opts.ProxyBaseUrl
	apiClient := proxyClientApi.NewAPIClient(configuration)
	resp, r, err := apiClient.ConversionApi.GetPriceConversion(context.Background()).Amount(amount).SrcCurrency(srcCurrency).DstCurrency(dstCurrency).Mode(mode).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ConversionApi.GetPriceConversion``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	fmt.Fprintf(os.Stdout, "Response from `ConversionApi.GetPriceConversion`: %v\n", resp)
	return resp.Price
}
