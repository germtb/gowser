# gowser

A terminal-based web browser written in Go. Renders web pages as readable markdown with optional interactive navigation, JavaScript rendering, and vim-style keybindings.

## Features

- Renders HTML as clean, readable markdown
- Interactive TUI mode with navigable links
- JavaScript rendering via headless Chrome (for JS-heavy sites)
- Lite mode for text-friendly site alternatives
- Vim-style navigation
- In-page search with highlighting
- Save pages as markdown files
- DuckDuckGo search integration
- Readability mode (extracts main content) with raw mode toggle

## Requirements

- Go 1.21+
- [gox](https://github.com/germtb/gox) - Go extension for JSX-like syntax
- Chrome/Chromium (optional, for `-j` JavaScript mode)

## Installation

```bash
# Install gox transpiler
go install github.com/germtb/gox@latest

# Clone the repository
git clone https://github.com/germtb/gowser.git
cd gowser

# Build
gox build .
```

## Usage

```bash
gowser [options] <url or search query>
```

### Options

| Flag | Description |
|------|-------------|
| `-i` | Interactive mode with navigable links (TUI) |
| `-s` | Search with DuckDuckGo |
| `-l`, `-lite` | Lite mode: use text-friendly site versions |
| `-j`, `-js` | JS mode: render JavaScript via headless browser |

Flags can be combined: `-il`, `-sil`, `-ij`, `-sij`

### Examples

```bash
# Simple fetch and display as markdown
gowser example.com

# Interactive browsing with TUI
gowser -i example.com

# Use lite/text-friendly versions
gowser -l reddit.com          # Uses old.reddit.com

# Render JavaScript-heavy sites
gowser -j instagram.com

# Interactive + JS rendering
gowser -ij react-app.com

# DuckDuckGo search
gowser -s cli web browser

# Search in interactive mode
gowser -si golang tutorial
```

### Lite Mode Mappings

| Original | Lite Version |
|----------|--------------|
| reddit.com | old.reddit.com |
| twitter.com / x.com | nitter.net |
| cnn.com | lite.cnn.com |
| npr.org | text.npr.org |
| wikipedia.org | mobile version |

## Keybindings (Interactive Mode)

| Key | Action |
|-----|--------|
| `q` | Quit |
| `r` | Reload page |
| `y` | Copy current URL to clipboard |
| `t` | Toggle readability/raw mode |
| `/` | Open search overlay |
| `n` / `N` | Next/previous search result |
| `Enter` | Follow selected link / submit |
| `s` | Save page as markdown |
| `b` / `f` | Back / forward in history |
| `g` / `G` | Go to top / bottom |
| `j` / `k` | Scroll down / up |
| `Tab` / `Shift+Tab` | Next / previous link |
| `Esc` | Close overlay / clear search |

## Architecture

```
gowser/
├── main.gox              # Main TUI application (gox/JSX syntax)
└── internal/
    ├── fetch/fetch.go    # HTTP fetching + headless browser (rod)
    ├── convert/convert.go # HTML to markdown conversion
    └── parser/parser.go  # Markdown parsing for TUI rendering
```

### Dependencies

- [goli](https://github.com/germtb/goli) - TUI framework
- [gox](https://github.com/germtb/gox) - JSX-like syntax for Go
- [go-readability](https://github.com/go-shiori/go-readability) - Content extraction
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) - HTML conversion
- [rod](https://github.com/go-rod/rod) - Headless browser automation

## License

MIT License - see [LICENSE](LICENSE) for details.
