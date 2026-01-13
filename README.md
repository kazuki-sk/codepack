# Code Packer (codepack)

🌐 **Languages:** [English](./README.md) | [日本語 (Japanese)](./README_ja.md)

---

**Code Packer (codepack)** is a CLI tool designed to consolidate an entire codebase into a single, LLM-friendly Markdown file. It automates the process of gathering source code while intelligently filtering out noise, maximizing the value of your LLM's context window.

---

## 🚀 Features

* **LLM-Optimized Output:** Generates a structured Markdown file containing your directory tree and file contents with appropriate syntax highlighting.
* **Token Efficiency:** Automatically excludes binaries, dependencies (like `node_modules`), and hidden files. It honors `.gitignore` and `.dockerignore` by default.
* **Performance First:** Built with a **Streaming-First** architecture. It processes large projects with minimal memory footprint using `io.Reader/Writer` pipelines.
* **Smart Language Detection:** Maps file extensions to programming languages for correct Markdown code blocks.
* **Clipboard Integration:** Use the `-c` flag to copy the output directly to your clipboard.
* **Safety & Control:** Detects large files (>500KB) and prompts for confirmation, or allows automated handling via `--force-large` or `--skip-large` flags.

---

## 📦 Installation

### For Go Users

```bash
go install github.com/kazuki-sk/codepack/cmd/codepack@latest

```

### For Non-Go Users (Binary)

1. Download the latest binary for your OS and architecture from the [Releases](https://github.com/kazuki-sk/codepack/releases) page.
2. Unzip/Untar the downloaded file.
3. Move the `codepack` binary to a directory in your system's `PATH` (e.g., `/usr/local/bin` for macOS/Linux).
4. (macOS/Linux) Grant execution permission: `chmod +x /usr/local/bin/codepack`.

---

## 🛠 Usage

### Basic Usage

Pack the current directory into `codebase.md`:

```bash
codepack -o codebase.md

```

### Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `-d` | string | `.` | Target directory to scan. |
| `-o` | string | `codebase.md` | Output Markdown file name. |
| `-c` | bool | `false` | Copy output to clipboard. |
| `-i` | strings | `[]` | Path to additional ignore files. |
| `-p` | strings | `[]` | Additional ignore patterns (e.g., `-p "*.log"`). |
| `-m` | string | `""` | Path to a custom language map JSON file. |
| --force-large | bool | false | Include large files without confirmation. |
| --skip-large | bool | false | Skip large files without confirmation. |
| -v, --version | bool | false | Show version information. |

---

## 📝 Output Example

`codepack` generates content like this, perfect for direct consumption by ChatGPT or Claude:

```markdown
# Project: my-app

## Directory Tree
- cmd/
  - main.go
- internal/
  - config.go

---
## File: cmd/main.go
```go
package main
func main() { ... }
```

## File: internal/config.go
```go
package config
type Config struct { ... }
```

```

---

## 🔍 Configuration

### Custom Language Map (`-m`)

`codepack` supports custom language mappings via a JSON file. The format follows the [LinguistMap](https://github.com/gusanmaz/LinguistMap) structure, where values are arrays of strings:

```json
{
  ".go": ["go", "golang"],
  ".json": ["json"],
  ".vue": ["vue"],
  ".myext": ["python"]
}

```

### Ignore Rules Priority

Rules are applied in this order:

1. Built-in Default Rules (Binaries, etc.)
2. CLI Patterns (`-p`)
3. CLI Files (`-i`)
4. Local `.code-packignore`
5. Standard `.gitignore` and `.dockerignore`

---
