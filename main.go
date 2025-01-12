package main

import (
	"bufio"
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

type config struct {
	APIToken string `yaml:"api_token"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func main() {
	fromLang := flag.String("f", "en", "Source language")
	flag.StringVar(fromLang, "from", "en", "Source language")
	toLang := flag.String("t", "en", "Target language")
	flag.StringVar(toLang, "to", "en", "Target language")
	apiToken := flag.String("a", "", "OpenAI API token")
	copyOutput := flag.Bool("c", true, "Copy output to clipboard")
	flag.BoolVar(copyOutput, "copy", true, "Copy output to clipboard")
	verbose := flag.Bool("verbose", false, "Enable verbose mode")
	version := flag.Bool("version", false, "Show version")
	flag.BoolVar(version, "v", false, "Show version")
	flag.Usage = usage

	flag.Parse()

	if *version {
		fmt.Println("Version:", Version)
		os.Exit(0)
	}

	if *verbose {
		log.Println("Verbose mode enabled.")
	}

	args := flag.Args()

	var textToTranslate string

	if flag.NArg() > 0 {
		textToTranslate = strings.Join(args, " ")
	} else {
		info, err := os.Stdin.Stat()
		if err != nil {
			fmt.Println("Error reading stdin:", err)
			os.Exit(1)
		}

		if (info.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			var input []string
			for scanner.Scan() {
				input = append(input, scanner.Text())
			}
			textToTranslate = strings.Join(input, " ")
		}
	}

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

	translatedText, err := translate(token, *fromLang, *toLang, textToTranslate, *verbose)
	if err != nil {
		log.Fatalf("Translation failed: %v", err)
	}

	fmt.Println(translatedText)

	if *copyOutput {
		if err := clipboard.WriteAll(translatedText); err != nil {
			log.Fatalf("Failed to copy translation to clipboard: %v", err)
		}
	}
}

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

	var config config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.APIToken == "" {
		return "", fmt.Errorf("API token not found in config file")
	}

	return config.APIToken, nil
}

func translate(apiKey, fromLang, toLang, text string, verbose bool) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	messages := []message{
		{
			Role:    "system",
			Content: "You are a translator that only gives the translated text",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Translate this from %s to %s: %s", fromLang, toLang, text),
		},
	}

	chatRequest := chatRequest{
		Model:    "gpt-4o-mini",
		Messages: messages,
	}

	jsonPayload, err := json.Marshal(chatRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	if verbose {
		log.Printf("Request Payload: %s\n", string(jsonPayload))
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create new HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

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

	var chatResponse chatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal response JSON: %w", err)
	}

	if chatResponse.Error.Message != "" {
		if verbose {
			log.Printf("OpenAI API Error: %s (Type: %s)\n", chatResponse.Error.Message, chatResponse.Error.Type)
		}
		return "", errors.New(chatResponse.Error.Message)
	}

	if len(chatResponse.Choices) > 0 {
		return strings.TrimSpace(chatResponse.Choices[0].Message.Content), nil
	}

	return "", errors.New("no translation found in response")
}

func usage() {
	msg := `
Usage:
  translate-cli [options] <text>

Options:
  -f, -from <language>      Source language (default: en)
  -t, -to   <language>      Target language (default: en)
  -a        <api token>     OpenAI API token (default: ~/.config/openai/secret.yml)
  -c, -copy                 Copy output to clipboard (default: true)
  -verbose                  Enable verbose mode (default: false)
  -v, -version              Show version
  -help                     Show this help message

Examples:
  translate-cli -f en -t es "Hello, how are you?"
  echo "Hello, how are you?" | translate-cli -f en -t es

OpenAI API Token:
  You can obtain an OpenAI API token from https://platform.openai.com/api-keys.
  
  The config file should be a YAML file with the following structure:
    api_token: YOUR_API_TOKEN

  And should be located at ~/.config/openai/secret.yml
`
	fmt.Fprintln(os.Stdout, msg)
}
