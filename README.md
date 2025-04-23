# Sizer

Sizer is a fast, efficient directory size analyzer written in Go. It provides a clear overview of storage usage by displaying the size of files and directories in a human-readable format. The tool is run using the command `sz`.

## Project Structure

The project follows the standard Go convention for command-line tools:

```
sizer/
├── cmd/
│   └── sz/         # Binary will be named 'sz'
│       └── main.go # Main application code
├── go.mod
├── LICENSE
└── README.md
```

## Features

- 📊 Lists files and directories sorted by size (largest first)
- 🔍 Human-readable file sizes (KB, MB, GB, etc.)
- ⚡ Parallel directory scanning for improved performance
- 🔄 Real-time progress updates during scanning
- 📁 Option to include hidden files (dotfiles)
- 📝 Shows total size and file count for scanned directories

## Installation

### Using Go

If you have Go installed (requires Go 1.24+):

```bash
go install github.com/eli-rich/sizer/cmd/sz@latest
```

### From Source

```bash
git clone https://github.com/eli-rich/sizer.git
cd sizer
go build -o sz ./cmd/sz
```

## Usage

Basic usage:

```bash
sz [path] [-a]
```

- `[path]`: Directory to analyze (defaults to current directory if not specified)
- `-a`: Include hidden files and directories (those starting with a dot)

Examples:

```bash
# Analyze current directory
sz

# Analyze a specific directory
sz /path/to/directory

# Include hidden files
sz -a

# Analyze a specific directory including hidden files
sz /path/to/directory -a
```

## Output Example

```
Contents of: /home/user/documents
----------------------------------------
DIR    215.5 MB        projects
DIR    125.3 MB        photos
FILE   25.2 MB         archive.zip
FILE   5.3 MB          report.pdf
FILE   125.0 KB        notes.txt
----------------------------------------
TOTAL: 371.5 MB (324 files)
```

## How It Works

Sizer reads the specified directory and:

1. Immediately processes files at the root level
2. Creates a worker pool to scan subdirectories in parallel
3. Shows real-time scanning progress
4. Aggregates and sorts results by size
5. Displays a summary with total size and file count

## License

This project is licensed under the terms found in the [LICENSE](LICENSE) file.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.
