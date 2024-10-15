# CLI Translator

🌍 A simple CLI application for translations using OpenAI API.

## Features

- Translates text between different languages using OpenAI's chat models.
- Command-line flags to set the source language, target language, and API key.
- Automatically copies the translation to your clipboard (optional).
- Verbose mode for detailed output during execution.

## Requirements

- [Go](https://golang.org/doc/install) 1.16 or higher.
- An OpenAI API key. You can generate one [here](https://platform.openai.com/api-keys).

## Installation

You can install the CLI Translator directly using the `go install` command:

```bash
go install github.com/hvpaiva/translate-cli@latest
```

> [!Note]
> This will install the application and make it available in your system's `$GOPATH/bin`, so ensure that your `$GOPATH/bin` is included in your system's `PATH`.

## Setting up the OpenAI API Key

Before you can use the app, you need to provide your OpenAI API key. You can do this in one of two ways:

1. **Using a configuration file**:

   To create a configuration file and add your OpenAI API token, you can use the following command:

   ```bash
   mkdir -p ~/.config/openapi && echo "api_token: your_openai_api_key_here" > ~/.config/openapi/secret.yml
   ```

   Replace `your_openai_api_key_here` with your actual API key. This will create the necessary directory and file in one go.

2. **Using a command-line flag**:

   You can also provide the API key directly via the command line using the `-a` flag:

   ```bash
   translate-cli -a your_openai_api_key_here "Text to translate"
   ```

## Usage

```bash
translate-cli [options] "Text to translate"
```

### Options:

- `-f` : Source language (default: `en`).
- `-t` : Target language (default: `en`).
- `-a` : OpenAI API token (optional if provided in the config file).
- `-cp` : Copy output to clipboard (default: `true`).
- `-v` : Enable verbose mode for more detailed logs (default: false).

### Example:

To translate a sentence from English to Spanish:

```bash
translate-cli -f en -t es "Hello, how are you?"
```

To translate with verbose mode enabled:

```bash
translate-cli -v -f en -t fr "Good morning"
```

## Troubleshooting

If the app fails to find the API token, ensure:

- The configuration file exists at `~/.config/openapi/secret.yml`.
- The file is correctly formatted in YAML.

You can also check the verbose output using the `-v` flag to diagnose any issues.

## License

This project is licensed under the MIT License.
