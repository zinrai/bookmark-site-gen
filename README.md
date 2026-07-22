# bookmark-site-gen

A static site generator that creates a thumbnail gallery from a list of bookmarked URLs.

## Features

- Generates screenshot thumbnails for each bookmarked URL
- Produces a single `index.html` with a responsive grid layout
- Idempotent operation: only creates new thumbnails and removes orphaned ones
- JSON input format

## Requirements

Chrome or Chromium

## Usage

```bash
$ bookmark-site-gen [options] <bookmarks.json>
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `-output` | `public` | Output directory |
| `-timeout` | `10s` | Screenshot capture timeout |
| `-dry-run` | false | Show changes without applying |
| `-no-sandbox` | false | Disable Chrome sandbox (for CI environments) |

### Example

Basic usage

```bash
$ bookmark-site-gen bookmarks.json
```

Specify output directory

```bash
$ bookmark-site-gen -output dist bookmarks.json
```

Preview changes without execution

```bash
$ bookmark-site-gen -dry-run bookmarks.json
```

## Input Format

`bookmarks.json` is a array of URLs:

```json
[
  "https://example.com/article1",
  "https://example.com/article2",
  "https://example.org/tool"
]
```

## Output

```
public/
├── index.html
└── thumbnails/
    ├── a1b2c3d4....png
    ├── b2c3d4e5....png
    └── c3d4e5f6....png
```

Thumbnail filenames are SHA-256 hashes of the URLs, ensuring idempotent operation.

## License

This project is licensed under the [MIT License](./LICENSE).
