package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"gopkg.in/yaml.v2"
)

// Config represents the structure for reading the OpenAI API token from a YAML file.
type Config struct {
	APIToken string `yaml:"api_token"`
}

// Message represents a single message in the OpenAI API chat request.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the full request body for the OpenAI API chat completion.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// ChatResponse represents the response structure from the OpenAI API.
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func main() {
	// Command-line flags
	fromLang := flag.String("f", "en", "Source language (default: en)")
	toLang := flag.String("t", "en", "Target language (default: en)")
	apiToken := flag.String("a", "", "OpenAI API token (optional)")
	copyOutput := flag.Bool("cp", true, "Copy output to clipboard (default: true)")
	verbose := flag.Bool("v", false, "Enable verbose mode (default: false)")

	flag.Parse()

	if *verbose {
		log.Println("Verbose mode enabled.")
	}

	// Ensure there's text to translate
	if flag.NArg() == 0 {
		log.Fatal("No text provided for translation.")
	}
	textToTranslate := flag.Arg(0)

	// Retrieve the API token either from the flag or the config file
	var token string
	if *apiToken == "" {
		var err error
		token, err = loadTokenFromConfig()
		if err != nil {
			log.Fatalf("Failed to load API token: %v", err)
		}
	} else {
		token = *apiToken
	}

	if *verbose {
		log.Printf("OpenAI Token: %s\n", token)
	}

	// Perform the translation
	translatedText, err := translate(token, *fromLang, *toLang, textToTranslate, *verbose)
	if err != nil {
		log.Fatalf("Translation failed: %v", err)
	}

	// Output the translation
	fmt.Println(translatedText)

	// Copy the translation to the clipboard if the flag is enabled
	if *copyOutput {
		if err := clipboard.WriteAll(translatedText); err != nil {
			log.Fatalf("Failed to copy translation to clipboard: %v", err)
		}
		fmt.Println("(Translation copied to clipboard)")
	}
}

// loadTokenFromConfig attempts to load the OpenAI API token from the user's config file.
func loadTokenFromConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "openapi", "secret.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.APIToken == "" {
		return "", fmt.Errorf("API token not found in config file")
	}

	return config.APIToken, nil
}

// translate performs a request to the OpenAI API to translate the given text from one language to another.
func translate(apiKey, fromLang, toLang, text string, verbose bool) (string, error) {
	// Change the URL to the correct endpoint for chat models
	url := "https://api.openai.com/v1/chat/completions"

	// Constructing the message to be sent
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a translator that only gives the translated text",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Translate this from %s to %s: %s", fromLang, toLang, text),
		},
	}

	// Creating the request body
	chatRequest := ChatRequest{
		Model:    "gpt-4o-mini", // Make sure this is a valid model
		Messages: messages,
	}

	// Marshal the request body into JSON
	jsonPayload, err := json.Marshal(chatRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	if verbose {
		log.Printf("Request Payload: %s\n", string(jsonPayload))
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create new HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-OK responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if verbose {
			log.Printf("Response Status: %s\n", resp.Status)
			log.Printf("Response Headers: %v\n", resp.Header)
			log.Printf("Response Body: %s\n", string(body))
		}
		return "", fmt.Errorf("unexpected status code from OpenAI API: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal the response into the ChatResponse struct
	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response JSON: %w", err)
	}

	// Check if there's an error in the response
	if chatResponse.Error.Message != "" {
		if verbose {
			log.Printf("OpenAI API Error: %s (Type: %s)\n", chatResponse.Error.Message, chatResponse.Error.Type)
		}
		return "", errors.New(chatResponse.Error.Message)
	}

	// Extract the translated text
	if len(chatResponse.Choices) > 0 {
		return strings.TrimSpace(chatResponse.Choices[0].Message.Content), nil
	}

	return "", errors.New("no translation found in response")
}
