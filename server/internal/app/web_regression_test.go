package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapperSettingsFocusDoesNotSuspendTelemetry(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "if(state.view==='mapper'&&$('#mapper-form').contains(document.activeElement))return;") {
		t.Fatal("editing Mapper settings must not suspend receiver progress or result polling")
	}
}

func TestTransmitLabExposesGroundedFixtureControls(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="transmit-source"`, `value="fixture">Known test fixture`, `id="fixture-kind"`, `value="qpsk"`, `id="fixture-snr"`, `id="fixture-offset"`, `id="fixture-iq-phase"`, `id="fixture-result"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("fixture control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"fixtureMode", "frequencyOffsetHz", "measuredEVMPercent", "decodeStatus"} {
		if !strings.Contains(app, required) {
			t.Fatalf("fixture behavior %q is missing", required)
		}
	}
}

func TestExpertTransmitModeRemovesOnlyRepeatedAcknowledgement(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="expert-transmit-toggle"`, `I know what I’m doing`, `does not bypass local-only access`, `60-second transmission ceiling`} {
		if !strings.Contains(index, required) {
			t.Fatalf("expert-mode disclosure %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{`gpsdr-expert-transmit-v1`, `Enable expert transmit mode?`, `if(expertTransmitMode&&!$('#transmit-dry-run').checked)$('#transmit-armed').checked=true`} {
		if !strings.Contains(app, required) {
			t.Fatalf("expert-mode behavior %q is missing", required)
		}
	}
	backendData, err := os.ReadFile("transmit.go")
	if err != nil {
		t.Fatal(err)
	}
	backend := string(backendData)
	for _, required := range []string{`request.DurationSecond > 60`, `!request.DryRun && !request.Armed`, `device.FirmwareSelfTestWarning`} {
		if !strings.Contains(backend, required) {
			t.Fatalf("non-bypassable transmit guard %q is missing", required)
		}
	}
	httpData, err := os.ReadFile("http.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(httpData), `if !requestIsLocal(r)`) {
		t.Fatal("local-computer transmit restriction is missing")
	}
}

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

func TestStandaloneAnalyzerUsesNativeSweepAndClampsFullRange(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if !strings.Contains(source, "full?Number(device.frequencyMinimumHz)") || !strings.Contains(source, "full?Number(device.frequencyMaximumHz)") {
		t.Fatal("full-range analyzer start must use each receiver's reported tuning limits")
	}
	if !strings.Contains(source, "/api/spectrum-analyzer/start") || !strings.Contains(source, "deviceIDs:ranges.map(item=>item.device.id)") {
		t.Fatal("analyzer must use the native multi-receiver sweep endpoint")
	}
}

func TestBandMonitorIncludesLiveWidebandDisplay(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="band-spectrum"`, `id="band-waterfall"`, `id="band-spectrum-cursor"`, `id="band-applied-state"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("Band Monitor display %q is missing", required)
		}
	}

	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"drawSpectrumCanvas($('#band-spectrum'))", "drawWaterfallCanvas(canvas)", "selectedDevice=connected.find", "Applied receiver state", "telemetryMatches"} {
		if !strings.Contains(app, required) {
			t.Fatalf("Band Monitor behavior %q is missing", required)
		}
	}
}

func TestLiveOverviewKeepsSessionControlsAndPromotesDecoderWorkspaces(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="live-task-overview"`, `id="live-quick-controls"`, `data-decoder-id="p25"`, `data-decoder-id="dsd-fme"`, `id="view-relay"`, `id="relay-start"`, `id="relay-streams"`, `id="relay-audio-device"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("Live/decoder/relay UI %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{`function renderLiveOverview()`, `data-live-task-action="stop"`, `api('/api/mapper/jobs/stop'`, `state.view==='mapper'||state.view==='analyzer'||state.view==='live'`} {
		if !strings.Contains(app, required) {
			t.Fatalf("Live overview behavior %q is missing", required)
		}
	}
}

func TestPiPowerHatCardIsCapabilityDriven(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"state.status?.powerHat", "PiPower5 power HAT", "renderPiPowerHatStatus()"} {
		if !strings.Contains(app, required) {
			t.Fatalf("Pi power HAT behavior %q is missing", required)
		}
	}
}

func TestHardwareActivityNamesOnlyTheAssignedReceiver(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"const p25Receivers=state.integrations?.p25?.receiverDeviceIDs||[]",
		"const telemetryDeviceID=state.status?.receiverTelemetry?.deviceID",
		"p25Receivers.includes(device.id)||telemetryDeviceID===device.id",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("hardware activity attribution %q is missing", required)
		}
	}
	if strings.Contains(app, "if(device.connected&&state.status?.running)return `Streaming · ${state.status.mode}`;") {
		t.Fatal("hardware page still labels every connected receiver as streaming")
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

func TestNativeUpdaterVerifiesPackageAndPreservesUserData(t *testing.T) {
	projectRoot := filepath.Join("..", "..", "..")
	shellData, err := os.ReadFile(filepath.Join(projectRoot, "macos", "GPSDRApp.m"))
	if err != nil {
		t.Fatal(err)
	}
	shell := string(shellData)
	for _, required := range []string{
		"api.github.com/repos/DragonKeeperAlex/GP-SDR/releases/latest",
		"-macos-universal.zip",
		"SHA256SUMS.txt",
		"sha256ForFile",
		`@"--verify", @"--deep", @"--strict"`,
		`@"app.gp-sdr.desktop"`,
		`window.gpsdrNativeCapabilities.push('appUpdater')`,
	} {
		if !strings.Contains(shell, required) {
			t.Fatalf("native updater safety behavior %q is missing", required)
		}
	}
	if strings.Contains(shell, "Application Support/GP-SDR") || strings.Contains(shell, "mapper-records.json") {
		t.Fatal("native updater must not manipulate GP-SDR user data")
	}
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`id="app-update-button"`, `id="app-update-auto"`, `id="app-update-notes"`} {
		if !strings.Contains(string(indexData), required) {
			t.Fatalf("update interface %q is missing", required)
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
		`id="mapper-concurrent"`, `256 · fast collection`, `512 · maximum collection`, `1,024 · extreme collection`,
		`id="mapper-active-job-list"`, `id="mapper-rf-grid"`, `Every active Mapper receiver is shown at the same time`,
		`id="mapper-results-toggle"`, `id="mapper-results-content"`, `id="mapper-filter-type"`, `id="mapper-filter-state"`, `id="mapper-sort"`, `id="mapper-filter-reset"`,
		`id="mapper-filter-repeated"`, `value="verified">Successfully identified`, `id="mapper-upload-verified"`, `Identified only`,
		`id="mapper-identify-min-hits"`, `id="mapper-identify-hit-source"`, `id="mapper-identify-occupancy"`, `100% only`,
		`class="mapper-tuning-panel"`, `id="mixer-search"`, `id="mixer-sort"`, `Active first`,
		`60 Hz · near real time`, `3× · performance`, `4096 bins · performance`,
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
		"renderMapperRF", "state.spectra", "mapper-active-row", "setMapperResultsCollapsed", "mapper-filter-type", "mapper-filter-state", "mapper-sort", "identifyMinimumHits", "identifyMinimumOccupancy", "mapper-filter-repeated", "mapperFullyIdentified", "uploadVerifiedOnly", "visibleRecords=records.slice(0,250)", "mapper-results-more", "/api/mapper/jobs/start-all", "receiverLabel",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("Mapper live or expandable-detail behavior %q is missing", required)
		}
	}
	for _, required := range []string{"renderSpectrumAnalyzer", "startSpectrumAnalyzer", "analyzer-full-range", "saveAnalyzerCSV", "analyzer-clear", "analyzer-reset-view", "/api/spectrum-analyzer/start", "/api/spectrum-analyzer/stop", "analyzer-instrument", "Spectrum bins", "scroll to zoom", "pointermove"} {
		if !strings.Contains(app, required) && !strings.Contains(index, required) {
			t.Fatalf("Spectrum Analyzer behavior %q is missing", required)
		}
	}
}

func TestSettingsExposeBoundedCaptureStorageControls(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexData)
	for _, required := range []string{`id="storage-policy-form"`, `id="storage-auto-cleanup"`, `id="storage-max-days"`, `id="storage-recording-cap"`, `id="storage-iq-cap"`, `id="storage-capture-journal-cap"`, `id="storage-clean-now"`, `Results are separate.`, `id="display-fps"`, `value="60">60 Hz`} {
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
	if !strings.Contains(app, "Math.max(16,1000/displayPrefs.fps)") {
		t.Fatal("spectrum refresh is still capped below the 60 Hz setting")
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
	for _, required := range []string{`id="tuner-hardware-center"`, `id="tuner-frequency"`, `id="tuner-lock-center"`, `id="tuner-cursor"`, `id="display-peak-hold"`, `class="radio-controls tuner-grouped-controls"`, `class="tuner-digit-readout"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("native tuner control %q is missing", required)
		}
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"hardwareCenterHz:", "lockCenter:", "setVFOFromPointer", "setTunerListenFrequency", "renderTunerDigits", "preserveCenter:active", "hardware center held", "Software VFO", "spectrumPeaks", "dataset.pending='true'", "dataset.pending !== 'true'"} {
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
	for _, required := range []string{"gpsdr-master-audio-v1", "liveAudio.masterGain", "recordingPlayer.muted", "applyMasterAudio", "void pumpLiveAudio(controller)", "maximumBacklog=.75", "Audio reconnecting", "Live audio interrupted; reconnecting"} {
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
		"right.eventCount", "rightTime", "item.lastHeardAt", "item.eventCount", "status.receiverDeviceIDs", "const liveDeviceID=wasRunning?(state.p25Status?.receiverDeviceIDs||[])[0]:''",
		"const p25Action=", "P25 searching…", "Decoder PCM", "status.audioFrames", "id=\"p25-audio-monitor\"", "P25 audio enabled",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("P25 mixer status or activity-order behavior %q is missing", required)
		}
	}
}

func TestReceiverSpecificControlsDoNotSendHackRFFrontEndSettingsToOtherRadios(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{
		"function updateReceiverCapabilityControls()", "setCapabilityVisibility(`${prefix}-${suffix}`,hackrf)",
		"lnaGainDB:hackrf?Number($('#tuner-lna').value):0", "vgaGainDB:hackrf?Number($('#tuner-vga').value):0",
		"ampEnabled:hackrf&&$('#tuner-amp').checked", "antennaPower:hackrf&&$('#tuner-bias').checked",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("receiver-specific control behavior %q is missing", required)
		}
	}
}

func TestTunerShowsEffectiveReceiverControls(t *testing.T) {
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appData)
	for _, required := range []string{"const appliedDevice=state.devices.find(item=>item.id===telemetry?.deviceID)", "Applied ${applied}", "LNA ${telemetry.lnaGainDB}"} {
		if !strings.Contains(app, required) {
			t.Fatalf("effective tuner-control readout %q is missing", required)
		}
	}
}

func TestUnifiedInterfaceKeepsControlsVisibleAndAutoContextual(t *testing.T) {
	indexData, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	appData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	cssData, err := os.ReadFile(filepath.Join("..", "..", "web", "app.css"))
	if err != nil {
		t.Fatal(err)
	}
	index, app, css := string(indexData), string(appData), string(cssData)
	for _, removed := range []string{`id="interface-mode"`, "gpsdr-interface-mode", "body:not(.advanced-mode)"} {
		if strings.Contains(index+app+css, removed) {
			t.Fatalf("legacy Beginner/Advanced interface behavior %q remains", removed)
		}
	}
	for _, required := range []string{`class="nav-group-label">Operate`, `class="radio-controls`, `id="mapper-gain-mode"`, `id="mapper-rate"`, `id="live-use-calibration"`} {
		if !strings.Contains(index, required) {
			t.Fatalf("unified interface control %q is missing", required)
		}
	}
	for _, required := range []string{"setContextControls", "auto-controlled", "#mapper-gain-mode"} {
		if !strings.Contains(app, required) && !strings.Contains(css, required) {
			t.Fatalf("contextual Auto behavior %q is missing", required)
		}
	}
}
