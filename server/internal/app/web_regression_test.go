package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHiddenSpectrumCanvasCannotCreateZeroIncrementLoop(t *testing.T) {
	path := filepath.Join("..", "..", "web", "app.js")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{
		"canvas.offsetParent === null",
		"width <= 0 || height <= 0",
		"const gridStepY = height / 5",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("spectrum regression guard %q is missing", required)
		}
	}
	if strings.Contains(source, "y += height / 5") {
		t.Fatal("zero-height canvas can still create an infinite loop")
	}
}

func TestLocationImportOffersValidatedCustomRange(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{
		`value="custom">Custom…`,
		`id="custom-range"`,
		`min="1" max="100"`,
	} {
		if !strings.Contains(index, required) {
			t.Fatalf("custom range control %q is missing", required)
		}
	}

	appPath := filepath.Join("..", "..", "web", "app.js")
	appData, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"selected === 'custom' ? Number($('#custom-range').value)",
		"radius < 1 || radius > 100",
		"'&radius=' + encodeURIComponent(radius)",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("custom range behavior %q is missing", required)
		}
	}
}
