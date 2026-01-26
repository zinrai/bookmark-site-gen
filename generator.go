package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Generator struct {
	OutputDir string
	Timeout   time.Duration
	DryRun    bool
}

func (g *Generator) Run(bookmarks []Bookmark) error {
	thumbnailDir := filepath.Join(g.OutputDir, "thumbnails")

	if err := g.ensureOutputDir(thumbnailDir); err != nil {
		return err
	}

	expected := g.buildExpectedMap(bookmarks)

	existing, err := g.listExistingThumbnails(thumbnailDir)
	if err != nil {
		return fmt.Errorf("list existing thumbnails: %w", err)
	}

	toCreate, toDelete := g.diff(expected, existing)

	g.deleteOrphanedThumbnails(thumbnailDir, toDelete)
	g.createThumbnails(thumbnailDir, toCreate, expected)

	return g.outputHTML(bookmarks)
}

func (g *Generator) ensureOutputDir(dir string) error {
	if g.DryRun {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return nil
}

func (g *Generator) buildExpectedMap(bookmarks []Bookmark) map[string]Bookmark {
	expected := make(map[string]Bookmark)
	for _, b := range bookmarks {
		expected[b.Hash] = b
	}
	return expected
}

func (g *Generator) deleteOrphanedThumbnails(dir string, hashes []string) {
	for _, hash := range hashes {
		path := filepath.Join(dir, hash+".png")
		if g.DryRun {
			fmt.Printf("Would delete: %s\n", path)
			continue
		}
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete %s: %v\n", path, err)
			continue
		}
		fmt.Printf("Deleted: %s\n", path)
	}
}

func (g *Generator) createThumbnails(dir string, hashes []string, expected map[string]Bookmark) {
	for _, hash := range hashes {
		b := expected[hash]
		path := filepath.Join(dir, hash+".png")

		if g.DryRun {
			fmt.Printf("Would create: %s (from %s)\n", path, b.URL)
			continue
		}

		fmt.Printf("Capturing: %s\n", b.URL)
		screenshot, err := CaptureScreenshot(b.URL, g.Timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to capture %s: %v\n", b.URL, err)
			continue
		}

		if err := os.WriteFile(path, screenshot, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", path, err)
			continue
		}
		fmt.Printf("Created: %s\n", path)
	}
}

func (g *Generator) outputHTML(bookmarks []Bookmark) error {
	if g.DryRun {
		fmt.Println("Would generate: index.html")
		return nil
	}

	if err := g.generateHTML(bookmarks); err != nil {
		return fmt.Errorf("generate html: %w", err)
	}
	fmt.Printf("Generated: %s\n", filepath.Join(g.OutputDir, "index.html"))
	return nil
}

func (g *Generator) listExistingThumbnails(dir string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return existing, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".png") {
			hash := strings.TrimSuffix(name, ".png")
			existing[hash] = struct{}{}
		}
	}

	return existing, nil
}

func (g *Generator) diff(expected map[string]Bookmark, existing map[string]struct{}) (toCreate, toDelete []string) {
	for hash := range expected {
		if _, ok := existing[hash]; !ok {
			toCreate = append(toCreate, hash)
		}
	}

	for hash := range existing {
		if _, ok := expected[hash]; !ok {
			toDelete = append(toDelete, hash)
		}
	}

	return toCreate, toDelete
}

func (g *Generator) generateHTML(bookmarks []Bookmark) error {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	path := filepath.Join(g.OutputDir, "index.html")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data := struct {
		Bookmarks []Bookmark
	}{
		Bookmarks: bookmarks,
	}

	return tmpl.Execute(f, data)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Bookmarks</title>
  <style>
    body {
      margin: 0;
      padding: 20px;
      background: #1a1a1a;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
      gap: 16px;
    }
    .grid a {
      display: block;
      border-radius: 8px;
      overflow: hidden;
      transition: transform 0.2s;
    }
    .grid a:hover {
      transform: scale(1.02);
    }
    .grid img {
      width: 100%;
      height: auto;
      display: block;
    }
  </style>
</head>
<body>
  <div class="grid">
    {{range .Bookmarks}}
    <a href="{{.URL}}" target="_blank" rel="noopener">
      <img src="thumbnails/{{.Hash}}.png" alt="" loading="lazy">
    </a>
    {{end}}
  </div>
</body>
</html>
`
