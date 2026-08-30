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

func TestMapperShowsDistinctDiscoveryAndIdentifyControls(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	if strings.Contains(index, ">Decipher<") {
		t.Fatal("Mapper still exposes the old Decipher workflow label")
	}
	for _, required := range []string{
		`id="mapper-current-frequency"`, `id="mapper-pass-progress"`, `id="mapper-progress-bar"`,
		`id="mapper-identified"`, `id="mapper-identified-detail"`, `id="mapper-eta"`, `id="mapper-eta-time"`,
		`id="mapper-workflow"`, `name="mapper-workflow" value="discovery"`, `name="mapper-workflow" value="decipher"`, `>Identify</span>`,
		`id="mapper-listen-value"`, `id="mapper-listen-unit"`, `value="86400">days`,
		`id="mapper-concurrent"`, `32 · highest CPU`,
		`id="mapper-operation"`, `id="mapper-channel-list"`, `id="mapper-spectrum"`, `id="mapper-waterfall"`,
		`id="mapper-results-toggle"`, `id="mapper-results-content"`, `id="mapper-filter-type"`, `id="mapper-filter-state"`, `id="mapper-sort"`, `id="mapper-filter-reset"`,
		`id="mapper-filter-repeated"`, `value="verified">Successfully identified`, `id="mapper-upload-verified"`, `Identified only`,
		`id="mapper-identify-min-hits"`, `id="mapper-identify-hit-source"`, `id="mapper-identify-occupancy"`, `100% only`,
		`class="mapper-tuning-panel"`, `id="mixer-search"`, `id="mixer-sort"`, `Active first`,
		`id="mapper-all-receivers"`, `Use all connected receivers`,
		`id="confirm-dialog"`, `id="confirm-dialog-message"`, `id="confirm-dialog-accept"`,
	} {
		if !strings.Contains(index, required) {
			t.Fatalf("Mapper status or Identify control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	if strings.Contains(app, "confirm(") {
		t.Fatal("Mapper still depends on the native browser confirmation API")
	}
	for _, required := range []string{
		"api('/api/mapper/progress')", "renderMapperProgress()", "mapperPeakHours", "mapperDetailHTML",
		"decipherListenSeconds", "concurrentChannels", "mapperBatchReadout", "expandedMapperFrequencies", "confirmAction", "Delete Mapper job?", "Clear Mapper results?",
		"renderMapperRF", "mapperSpectrumJob", "setMapperResultsCollapsed", "mapper-filter-type", "mapper-filter-state", "mapper-sort", "identifyMinimumHits", "identifyMinimumOccupancy", "mapper-filter-repeated", "mapperFullyIdentified", "uploadVerifiedOnly", "visibleRecords=records.slice(0,250)", "mapper-results-more", "/api/mapper/jobs/start-all", "receiverLabel",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("Mapper live or expandable-detail behavior %q is missing", required)
		}
	}
}

func TestSettingsExposeBoundedCaptureStorageControls(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="storage-policy-form"`, `id="storage-auto-cleanup"`, `id="storage-max-days"`, `id="storage-recording-cap"`, `id="storage-iq-cap"`, `id="storage-clean-now"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("storage control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"/api/storage/policy", "/api/storage/cleanup", "Storage limits saved", "Cleanup complete"} {
		if !strings.Contains(app, required) {
			t.Fatalf("storage behavior %q is missing", required)
		}
	}
}

func TestHardwareIncludesReceiverAndAntennaCharacterizationLab(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="characterization-form"`, `id="characterization-devices"`, `id="characterization-range-mode"`, `id="characterization-antenna-min"`, `id="characterization-points"`, `id="characterization-results"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("receiver characterization control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"renderCharacterization", "drawCharacterizationChart", "/api/calibrations/characterization/start", "Ambient comparison"} {
		if !strings.Contains(app, required) {
			t.Fatalf("receiver characterization behavior %q is missing", required)
		}
	}
}

func TestNativeTunerSeparatesHardwareCenterAndSoftwareVFO(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="tuner-hardware-center"`, `id="tuner-frequency"`, `id="tuner-lock-center"`, `id="tuner-cursor"`, `id="display-peak-hold"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("native tuner control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"hardwareCenterHz:", "lockCenter:", "setVFOFromPointer", "Software VFO", "spectrumPeaks", "dataset.pending='true'", "dataset.pending !== 'true'"} {
		if !strings.Contains(app, required) {
			t.Fatalf("native tuner behavior %q is missing", required)
		}
	}
}

func TestDMRControlsAreAvailableAcrossNativeWorkspaces(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="live-mode"`, `id="tuner-mode"`, `value="dmr"`, `id="mapper-decoder"`, `value="discovery"`, `value="decipher"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("DMR or Mapper control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"decoderForSelectedMode", "preferredDecoder:", "decoder-new-profile", "message.timeSlot", "message.colorCode"} {
		if !strings.Contains(app, required) {
			t.Fatalf("DMR behavior %q is missing", required)
		}
	}
}

func TestTopBarHasPersistentMasterAudioControls(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="master-mute"`, `id="master-volume"`, `id="master-volume-value"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("master audio control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"gpsdr-master-audio-v1", "liveAudio.masterGain", "recordingPlayer.muted", "applyMasterAudio", "void pumpLiveAudio(controller)"} {
		if !strings.Contains(app, required) {
			t.Fatalf("master audio behavior %q is missing", required)
		}
	}
}

func TestSignalIndicatorUsesMeasuredDetection(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appData), "receiving=!!telemetry?.signalDetected") {
		t.Fatal("signal indicator must not treat forced-open audio as RF detection")
	}
}

func TestMissingComponentDialogOffersInstallGuideAndIgnore(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`id="missing-components-dialog"`, `id="missing-components-ignore"`, `id="missing-components-review"`} {
		if !strings.Contains(string(indexData), required) {
			t.Fatalf("missing-component prompt %q is absent", required)
		}
	}
	for _, required := range []string{"renderMissingComponents", "gpsdr-ignored-components", "setupActions(item.id)"} {
		if !strings.Contains(string(appData), required) {
			t.Fatalf("missing-component behavior %q is absent", required)
		}
	}
}

func TestP25MixerShowsControlChannelAndActivityOrdering(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"status.controlChannelHz", `id="p25-order"`, "Most recent", "Most received",
		"right.eventCount", "rightTime", "item.lastHeardAt", "item.eventCount",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("P25 mixer status or activity-order behavior %q is missing", required)
		}
	}
}
