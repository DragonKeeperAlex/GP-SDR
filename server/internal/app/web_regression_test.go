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

func TestMapperLocationUsesNativeBridgeWithWebFallback(t *testing.T) {
	appPath := filepath.Join("..", "..", "web", "app.js")
	appData, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"gpsdrNativeCapabilities?.includes('location')",
		"native.postMessage({action:'requestLocation'})",
		"navigator.geolocation.getCurrentPosition",
		"window.gpsdrNativeLocationResult",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("Mapper location behavior %q is missing", required)
		}
	}

	projectRoot := filepath.Join("..", "..", "..")
	infoData, err := os.ReadFile(filepath.Join(projectRoot, "packaging", "macos", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(infoData), "NSLocationWhenInUseUsageDescription") {
		t.Fatal("the macOS bundle is missing its location usage description")
	}

	shellData, err := os.ReadFile(filepath.Join(projectRoot, "macos", "GPSDRApp.m"))
	if err != nil {
		t.Fatal(err)
	}
	shell := string(shellData)
	for _, required := range []string{
		"CLLocationManagerDelegate",
		"requestWhenInUseAuthorization",
		"window.gpsdrNativeLocationResult",
		"window.gpsdrNativeCapabilities=['location','localDatabaseFolder']",
	} {
		if !strings.Contains(shell, required) {
			t.Fatalf("native location bridge %q is missing", required)
		}
	}
}

func TestMapperShowsLiveProgressLongDecipherTimingAndExpandableEvidence(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{
		`id="mapper-current-frequency"`, `id="mapper-pass-progress"`, `id="mapper-progress-bar"`,
		`id="mapper-listen-value"`, `id="mapper-listen-unit"`, `value="86400">days`,
	} {
		if !strings.Contains(index, required) {
			t.Fatalf("Mapper status or decipher timing control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"api('/api/mapper/progress')", "renderMapperProgress()", "mapperPeakHours", "mapperDetailHTML",
		"decipherListenSeconds", "expandedMapperFrequencies",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("Mapper live or expandable-detail behavior %q is missing", required)
		}
	}
}
