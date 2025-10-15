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

// suspendWorkspace suspends the given workspace and waits for it to become SUSPENDED
func suspendWorkspace(client *management.ClientWithResponses, workspaceID management.WorkspaceID) error {
	fmt.Printf("Suspending workspace %s...\n", workspaceID)

	// Suspend the workspace
	var errR error
	maxIterations := 10
	for {
		if maxIterations <= 0 {
			return fmt.Errorf("failed to suspend workspace after multiple attempts: %s", errR)
		}

		maxIterations--

		resp, err := client.PostV1WorkspacesWorkspaceIDSuspendWithResponse(context.TODO(), workspaceID)
		if err != nil {
			return fmt.Errorf("failed to suspend workspace: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			time.Sleep(3 * time.Second)
			errR = fmt.Errorf("suspend request returned %s: %s", resp.Status(), resp.Body)
			continue
		}

		break
	}

	// Wait for workspace to become SUSPENDED
	for {
		statusResp, err := client.GetV1WorkspacesWorkspaceIDWithResponse(context.TODO(), workspaceID, &management.GetV1WorkspacesWorkspaceIDParams{})
		if err != nil {
			return fmt.Errorf("failed to get workspace status: %w", err)
		}

		if statusResp.StatusCode() != http.StatusOK {
			return fmt.Errorf("status request returned %s: %s", statusResp.Status(), statusResp.Body)
		}

		if statusResp.JSON200.State == management.WorkspaceStateSUSPENDED {
			fmt.Printf("Workspace %s is now SUSPENDED\n", workspaceID)
			break
		}

		fmt.Printf("Workspace status is %s, waiting to become SUSPENDED\n", statusResp.JSON200.State)
		time.Sleep(3 * time.Second)
	}

	return nil
}

// resumeWorkspace resumes the given workspace and waits for it to become ACTIVE
func resumeWorkspace(client *management.ClientWithResponses, workspaceID management.WorkspaceID) error {
	fmt.Printf("Resuming workspace %s...\n", workspaceID)

	// Resume the workspace
	maxIterations := 10
	var errR error
	for {
		if maxIterations <= 0 {
			return fmt.Errorf("failed to resume workspace after multiple attempts: %s", errR)
		}

		maxIterations--

		resp, err := client.PostV1WorkspacesWorkspaceIDResumeWithResponse(context.TODO(), workspaceID, management.PostV1WorkspacesWorkspaceIDResumeJSONRequestBody{})
		if err != nil {
			return fmt.Errorf("failed to resume workspace: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			errR = fmt.Errorf("resume request returned %s: %s", resp.Status(), resp.Body)
			time.Sleep(3 * time.Second)
			continue
		}

		break
	}

	// Wait for workspace to become ACTIVE
	for {
		statusResp, err := client.GetV1WorkspacesWorkspaceIDWithResponse(context.TODO(), workspaceID, &management.GetV1WorkspacesWorkspaceIDParams{})
		if err != nil {
			return fmt.Errorf("failed to get workspace status: %w", err)
		}

		if statusResp.StatusCode() != http.StatusOK {
			return fmt.Errorf("status request returned %s: %s", statusResp.Status(), statusResp.Body)
		}

		if statusResp.JSON200.State == management.WorkspaceStateACTIVE {
			fmt.Printf("Workspace %s is now ACTIVE\n", workspaceID)
			return nil
		}

		fmt.Printf("Workspace status is %s, waiting to become ACTIVE\n", statusResp.JSON200.State)
		time.Sleep(3 * time.Second)
	}
}

// testConnection tests the database connection by executing a simple query
func testConnection(endpoint, user, password string) error {
	connString := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?timeout=10s",
		user,
		password,
		endpoint,
		"information_schema",
	)

	dbConn, err := sqlx.ConnectContext(context.TODO(), "mysql", connString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbConn.Close()

	rows, err := dbConn.QueryContext(context.TODO(), "select 1")
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tmp int
		err := rows.Scan(&tmp)
		if err != nil {
			return fmt.Errorf("failed to scan result: %w", err)
		}
		fmt.Printf("Connection test successful: select 1 returned %d\n", tmp)
	}

	return nil
}

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
	regionName := "us-east-1"
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

	endpoint := *response.JSON200.Endpoint
	password = "StrongPassword132!"
	user := "admin"

	// Start suspend/resume loop with counter
	loopCount := 0
	for {
		loopCount++
		fmt.Printf("\n--- Starting loop iteration %d ---\n", loopCount)

		// Test connection before suspend
		fmt.Printf("Testing connection before suspend (iteration %d)...\n", loopCount)
		err := testConnection(endpoint, user, password)
		if err != nil {
			fmt.Printf("Connection test failed in iteration %d: %v\n", loopCount-1, err)
			fmt.Printf("Total successful loops completed: %d\n", loopCount-1)
			break
		}

		// Suspend the workspace
		err = suspendWorkspace(client, workspaceID)
		if err != nil {
			fmt.Printf("Failed to suspend workspace in iteration %d: %v\n", loopCount, err)
			fmt.Printf("Total successful loops completed: %d\n", loopCount-1)
			break
		}

		// Resume the workspace
		err = resumeWorkspace(client, workspaceID)
		if err != nil {
			fmt.Printf("Failed to resume workspace in iteration %d: %v\n", loopCount, err)
			fmt.Printf("Total successful loops completed: %d\n", loopCount-1)
			break
		}

		// Test connection after resume
		fmt.Printf("Testing connection after resume (iteration %d)...\n", loopCount)
		err = testConnection(endpoint, user, password)
		if err != nil {
			fmt.Printf("Connection test failed after resume in iteration %d: %v\n", loopCount, err)
			fmt.Printf("Total successful loops completed: %d\n", loopCount-1)
			break
		}

		fmt.Printf("Successfully completed loop iteration %d\n", loopCount)
	}
}
