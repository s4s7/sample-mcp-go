package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	qaEndpoint = "https://api-inference.huggingface.co/models/google/gemma-3-27b-it"
	hfToken    = "" // Hugging Face API token (replace with your actual token)
)

// main initializes the MCP server, registers the historical_events tool, and starts the HTTP server.
func main() {
	s := server.NewMCPServer("Historical Events", "1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Define a tool to fetch historical events for a given date.
	tool := mcp.NewTool("historical_events",
		mcp.WithDescription("Gets exactly 2 historical events that happened on a given date"),
		mcp.WithString("date",
			mcp.Required(),
			mcp.Description("Date in YYYY-MM-DD format"),
		),
	)
	s.AddTool(tool, historicalEventsHandler)

	// Start the server with Server-Sent Events (SSE) capabilities.
	sseServer := server.NewSSEServer(
		s,
		server.WithBaseURL("http://localhost:8000"),
		server.WithSSEEndpoint("/mcp/sse"),
		server.WithMessageEndpoint("/mcp"),
		server.WithHTTPServer(&http.Server{
			Addr: ":8000",
		}),
	)

	log.Println("Server running on :8000")
	log.Fatal(http.ListenAndServe(":8000", sseServer))
}

// historicalEventsHandler handles requests to fetch historical events for a given date.
func historicalEventsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var (
		date          string
		ok            bool
		err           error
		parsedDate    time.Time
		year          int
		monthDay      string
		prompt        string
		answer        string
		cleanedAnswer string
	)

	// Validate and parse the date parameter.
	if date, ok = req.Params.Arguments["date"].(string); !ok {
		return nil, fmt.Errorf("date must be a string")
	}

	parsedDate, err = time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, must be YYYY-MM-DD")
	}

	year = parsedDate.Year()
	monthDay = parsedDate.Format("January 2")

	// Construct the AI prompt to fetch historical events.
	prompt = fmt.Sprintf(`Provide exactly two significant historical events that happened on %s %d.
The events must have occurred on this exact date (same month and day).
Format your response exactly as:
1. [Year] Event description
2. [Year] Event description

If no events match this exact date, say "No significant historical events found for %s %d."`,
		monthDay, year, monthDay, year)

	// Query the Hugging Face model.
	if answer, err = queryGemma(prompt); err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}

	cleanedAnswer = strings.TrimSpace(answer)
	if strings.Contains(cleanedAnswer, "No significant historical events") {
		return mcp.NewToolResultText(fmt.Sprintf("No historical events found for %s %d", monthDay, year)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("On %s %d:\n%s", monthDay, year, cleanedAnswer)), nil
}

// queryGemma sends a request to the Hugging Face Gemma model to fetch historical events.
func queryGemma(prompt string) (string, error) {
	var (
		reqBody, respBody []byte
		err               error
		req               *http.Request
		client            *http.Client
		resp              *http.Response
		gemmaResponse     []struct {
			GeneratedText string `json:"generated_text"`
		}
	)

	// Prepare the request payload.
	if reqBody, err = json.Marshal(map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"max_new_tokens": 200,
			"temperature":    0.7,
		},
	}); err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Create an HTTP request.
	if req, err = http.NewRequest("POST", qaEndpoint, bytes.NewBuffer(reqBody)); err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+hfToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request.
	client = &http.Client{Timeout: 30 * time.Second}
	if resp, err = client.Do(req); err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read and parse the response body.
	if respBody, err = io.ReadAll(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if err = json.Unmarshal(respBody, &gemmaResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(gemmaResponse) == 0 || gemmaResponse[0].GeneratedText == "" {
		return "No historical events found for this date", nil
	}

	return gemmaResponse[0].GeneratedText, nil
}
