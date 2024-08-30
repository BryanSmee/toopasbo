package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/aws/aws-lambda-go/lambda"
	"smee.ovh/toopasbo/config"
	"smee.ovh/toopasbo/endpoints"
	"smee.ovh/toopasbo/transformers"
)

func HandleRequest(ctx context.Context, event interface{}) {
	fmt.Println("Running in job mode")
	var zipCode = "14700"
	var countryCode = "CZ"

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	is_running_in_aws := os.Getenv("RUNTIME_ENVIRONMENT") == "aws"
	var appConfig config.Config
	if is_running_in_aws {
		fmt.Println("Running in AWS")
		appConfig = config.NewConfigFromAWS()
	} else {
		fmt.Println("Running in local")
		appConfig = config.NewConfigFromEnv()
	}

	ctx = context.WithValue(ctx, config.AppConfigContextKey, appConfig)

	wc, err := endpoints.NewWebhookClient(ctx)
	if err != nil {
		panic(err)
	}

	var position, positionErr = wc.OpenWeatherClient.GetPosition(zipCode, countryCode)
	if positionErr != nil {
		fmt.Println(positionErr)
		os.Exit(1)
	}

	weather, err := wc.OpenWeatherClient.GetWeather(position)

	if err != nil {
		log.Fatal(err)
	}

	client, err := transformers.NewFalAIClient(ctx, "fal-ai/flux-pro")
	if err != nil {
		log.Fatal(err)
	}

	imageUrl, err := client.GenerateWeatherImage(weather)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(imageUrl)
	path, err := endpoints.DownloadFile(imageUrl)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(path)
	wc.SendToAll(ctx, path, endpoints.WeatherToTelegramText(weather))
}

func main() {
	lambda.Start(HandleRequest)
}
