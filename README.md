Here's a README for your Go CLI application:

---

# ifchange

`ifchange` is a lightweight CLI tool written in Go that watches a directory for file changes and runs a specified command whenever a change is detected. It's perfect for automating tasks like building and testing code whenever files are updated.

1. There are many hot loaders for various languages, Go doesnt have one.
2. Other hot reload CLI tools are really complex. this thing is just one file and has a few commands.

## Installation

To install `ifchange`, you can use the following one-liner:
(you need Golang installed.) [Install go here](https://go.dev/doc/install)

```bash
go install github.com/erikd234/ifchange@latest
```

Make sure your `GOPATH/bin` is in your `PATH` to easily run the `ifchange` command from anywhere.

## Usage

```bash
ifchange -dir <directory> -cmd "<command>" -only "<regex expression>"
```

- `-dir`: OPTIONAL: Specifies the directory to watch. Defaults to the current directory (`./`) if not provided.
- `-cmd`: The command to execute whenever a file change is detected.
- `-only`: OPTIONAL: Regex file match. If you only wanted to do .go files do `-only ".go"`.

### Example

```bash
ifchange -dir ./src -cmd "go test ./..."
```

This command will watch the `./src` directory for changes and run `go test ./...` whenever a file is modified.

## How it Works

- `ifchange` continuously monitors the specified directory and its subdirectories for any file changes.
- Upon detecting a change, it triggers the provided command.
- The tool runs in an infinite loop until it receives an interrupt signal (Ctrl+C), at which point it gracefully shuts down.

## Features

- **Lightweight:** Minimal dependencies and simple to use.
- **Cross-platform:** Works on Linux, macOS, and Windows.
- **Customizable:** Run any command in response to file changes.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests on the [GitHub repository](https://github.com/erikd234/ifchange).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

Let me know if you need any changes or additional sections!
