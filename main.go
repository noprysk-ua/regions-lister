package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/singlestore-labs/singlestore-go/management"

	_ "github.com/go-sql-driver/mysql"
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

	password := "StrongPassword132!"
	provider := management.AWS
	allowAllTraffic := true
	regionName := "eu-west-1"
	name := "NestorTestACTIVE"
	r, err := client.PostV1WorkspaceGroupsWithResponse(context.TODO(), management.PostV1WorkspaceGroupsJSONRequestBody{
		AdminPassword:   &password,
		AllowAllTraffic: &allowAllTraffic,
		FirewallRanges:  []string{},
		Provider:        &provider,
		RegionName:      &regionName,
		Name:            name,
	})
	if err != nil {
		log.Fatal(err)
	}

	if r.StatusCode() != http.StatusOK {
		log.Fatalf("request returned %s: %s", r.Status(), r.Body)
	}

	workspaceGroupID := r.JSON200.WorkspaceGroupID

	defer func() {
		force := true
		_, err := client.DeleteV1WorkspaceGroupsWorkspaceGroupIDWithResponse(context.TODO(), workspaceGroupID, &management.DeleteV1WorkspaceGroupsWorkspaceGroupIDParams{Force: &force})
		if err != nil {
			log.Printf("failed to delete workspace group %s: %v", workspaceGroupID, err)
		} else {
			log.Printf("workspace group %s deleted", workspaceGroupID)
		}
	}()

	// WORKSPACE NOW

	// workspaceGroupID := uuid.MustParse("b3a3a333-0f83-4c8e-b86c-f793d429f47a")
	resp, err := client.PostV1WorkspacesWithResponse(context.TODO(), management.PostV1WorkspacesJSONRequestBody{
		Name:             "nestor-test",
		WorkspaceGroupID: workspaceGroupID,
	})
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode() != http.StatusOK {
		log.Fatalf("request returned %s: %s", resp.Status(), resp.Body)
	}

	workspaceID := resp.JSON200.WorkspaceID

	defer func() {
		_, err := client.DeleteV1WorkspacesWorkspaceIDWithResponse(context.TODO(), workspaceID)
		if err != nil {
			log.Printf("failed to delete workspace %s: %v", workspaceID, err)
		} else {
			log.Printf("workspace %s deleted", workspaceID)
		}
	}()

	// TODO: To test manually, uncomment
	// workspaceID := uuid.MustParse("c1e1dfb9-f458-4ac1-a8c1-9642230edde5")

	var response *management.GetV1WorkspacesWorkspaceIDResponse
	for {
		var err error
		response, err = client.GetV1WorkspacesWorkspaceIDWithResponse(context.TODO(), workspaceID, &management.GetV1WorkspacesWorkspaceIDParams{})
		if err != nil {
			log.Fatal(err)
		}

		if response.StatusCode() != http.StatusOK {
			log.Fatalf("request returned %s: %s", response.Status(), response.Body)
		}

		if response.JSON200.State == management.WorkspaceStateACTIVE {
			break
		} else {
			fmt.Printf("Workspace status is %s, waiting to become ACTIVE\n", response.JSON200.State)
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Printf("Workspace %s is ACTIVE\n", response.JSON200.WorkspaceID)

	activeAt := time.Now()

	endpoint := *response.JSON200.Endpoint
	password = "StrongPassword132!"
	user := "admin"

	connString := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?timeout=3s", // With timeout
		user,
		password,
		endpoint,
		"information_schema",
	)

	dbConn, err := sqlx.ConnectContext(context.TODO(), "mysql", connString)
	if err != nil {
		log.Fatal(err)
	}

	defer dbConn.Close()

	rows, err := dbConn.QueryContext(context.TODO(), "select 1")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tmp int

		err := rows.Scan(&tmp)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("select 1 returned %d\n", tmp)
	}

	executedAt := time.Now()

	fmt.Printf("It took %d seconds between ACTIVE and EXECUTED\n", int(executedAt.Sub(activeAt).Seconds()))

	fmt.Printf("Workspace %s is ACTIVE\n", response.JSON200.WorkspaceID)
}
