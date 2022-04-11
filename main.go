package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/singlestore-labs/singlestore-go/management"
)

const apiServiceURL = "https://api.singlestore.com"

func main() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("Environmental variable $API_KEY should be set, visit https://docs.singlestore.com/managed-service/en/reference/management-api.html for details")
	}

	client, err := management.NewClientWithResponses(apiServiceURL,
		management.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	regions, err := client.GetV0betaRegionsWithResponse(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	if regions.HTTPResponse.StatusCode != http.StatusOK {
		log.Fatalf("request returned %s", regions.HTTPResponse.Status)
	}

	result, err := json.MarshalIndent(regions.JSON200, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", result)
}
