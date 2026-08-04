# pageshot

Captures a screenshot of a single web page.

## Features

- Waits for the page to stop loading before capturing, so a site that redirects through an interstitial is captured after the redirect
- Reports through the exit status whether the page settled within the time budget
- Leaves an existing output file alone unless asked to replace it
- Optional downscaling of the capture

## Requirements

Chrome or Chromium

## Usage

```bash
$ pageshot [options] <url>
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `-output` | | Output PNG path (required) |
| `-force` | false | Capture even if the output file already exists |
| `-timeout` | `10s` | Time budget for the page to settle |
| `-viewport` | `1280x800` | Browser viewport as `WIDTHxHEIGHT` |
| `-size` | | Resize the capture to `WIDTHxHEIGHT` |
| `-no-sandbox` | false | Disable Chrome sandbox (for CI environments) |

### Exit Status

| Code | Meaning |
|------|---------|
| 0 | Captured, or skipped because the output already exists |
| 1 | Failed |
| 2 | Captured before the page settled |

An exit status of 2 means the image is of a page that was still loading. Capture it again with a larger `-timeout` and `-force`.

### Example

Capture a page at the browser viewport size

```bash
$ pageshot -output page.png https://example.com/
```

Capture a thumbnail

```bash
$ pageshot -output thumbnail.png -size 400x250 https://example.com/
```

Replace an existing capture, giving a slow page more time

```bash
$ pageshot -output page.png -force -timeout 30s https://example.com/
```

## License

This project is licensed under the [MIT License](./LICENSE).
