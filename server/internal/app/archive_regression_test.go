package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistorySurvivesUpdatesBeyond25000AndRestart(t *testing.T) {
	root := t.TempDir()
	events := make([]TransmissionEvent, 25002)
	for i := range events {
		events[i] = TransmissionEvent{ID: fmt.Sprint(i), FrequencyHz: 155e6, StartedAt: time.Now(), AnalysisStatus: "pending"}
	}
	path := filepath.Join(root, "events.jsonl")
	if err := writeEventFile(path, events); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTranscript("0", "K6ABC testing preserved history"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("update rewrote original event history")
	}
	reopened, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	event, ok := reopened.Get("0")
	if reopened.Count() != len(events) || len(reopened.PendingAnalysis(0, "")) != len(events) || !ok || stringValue(event.Transcript) != "K6ABC testing preserved history" {
		t.Fatal("history or pending work lost across restart")
	}
}

func TestSpectrumDetectsBurstBetweenFormerSparseWindows(t *testing.T) {
	const rate = 1_000_000
	data := make([]byte, 512_000*2)
	seed := uint32(42)
	for n := 0; n < len(data)/2; n++ {
		seed = seed*1664525 + 1013904223
		i := float64(int(seed>>28) - 8)
		seed = seed*1664525 + 1013904223
		q := float64(int(seed>>28) - 8)
		if n >= 12000 && n < 20000 {
			phase := 2 * math.Pi * 100000 * float64(n) / rate
			i += 90 * math.Cos(phase)
			q += 90 * math.Sin(phase)
		}
		data[2*n] = byte(int8(i))
		data[2*n+1] = byte(int8(q))
	}
	levels, err := MeasureChannelSpectrum(data, ComplexSigned8, rate, 100e6, []ChannelDefinition{{ID: "burst", FrequencyHz: 100.1e6, BandwidthHz: 12500}, {ID: "quiet", FrequencyHz: 100.3e6, BandwidthHz: 12500}})
	if err != nil {
		t.Fatal(err)
	}
	if levels["burst"].SignalDB-levels["burst"].NoiseDB < 15 {
		t.Fatalf("missed burst: %+v", levels)
	}
	if levels["quiet"].SignalDB-levels["quiet"].NoiseDB > 6 {
		t.Fatalf("noise falsely marked active: %+v", levels)
	}
}

func TestChannelFIRRejectsAliasedSignalAndPreservesPassband(t *testing.T) {
	const rate = 2_000_000
	power := func(offset float64) float64 {
		data := make([]byte, rate/50*2)
		for n := 0; n < len(data)/2; n++ {
			phase := 2 * math.Pi * offset * float64(n) / rate
			data[2*n] = byte(int8(math.Round(100 * math.Cos(phase))))
			data[2*n+1] = byte(int8(math.Round(100 * math.Sin(phase))))
		}
		out, _, format := compactIQEvidence(data, ComplexSigned8, CaptureSpec{SampleRateHz: rate, CenterFrequencyHz: 100000000}, 100000000, 12500)
		sum := 0.0
		count := 0
		for n := 128; n < len(out)/2-128; n++ {
			i, q := decoderIQPair(out, n, format)
			sum += i*i + q*q
			count++
		}
		return sum / float64(count)
	}
	wanted, alias := power(8000), power(258000)
	if wanted < 8000 || 10*math.Log10(wanted/alias) < 35 {
		t.Fatalf("bad FIR response: passband=%f alias=%f", wanted, alias)
	}
}

func TestOriginalArchiveIsExactSharedAndProtected(t *testing.T) {
	root := t.TempDir()
	data := bytes.Repeat([]byte{0, 255, 128, 127}, 48000)
	spec := CaptureSpec{SampleRateHz: 32000, CenterFrequencyHz: 100000000}
	interval := CaptureInterval{ID: NewID(), ReceivedAt: time.Now().Add(-48 * time.Hour), SampleSeconds: 3, SampleBytes: len(data)}
	if err := writeCaptureInterval(root, &interval, spec, ComplexSigned8, data, true, 1<<30); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(interval.IQPath)
	if !bytes.Equal(saved, data) || interval.SHA256 == "" {
		t.Fatal("original samples changed")
	}
	for _, id := range []string{"a", "b"} {
		path, _, err := finalizeIQEvidence(interval.IQPath, TransmissionEvent{ID: id, IQRetentionPolicy: "delete"})
		if err != nil || path != interval.IQPath {
			t.Fatal("analysis moved/deleted shared original", err)
		}
	}
	enforceStoragePolicy(root, StoragePolicy{IQCapBytes: 1, MaxCaptureDays: 1}, time.Now())
	if !fileExists(interval.IQPath) {
		t.Fatal("cleanup deleted archive")
	}
	interval.ID = NewID()
	if err := writeCaptureInterval(root, &interval, spec, ComplexSigned8, data, true, 1); err == nil {
		t.Fatal("archive exceeded cap instead of stopping")
	}
}

func TestMediaRecoveryIsIdempotentAndFlagsMissingFiles(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(TransmissionEvent{ID: "missing", IQPath: ptr(filepath.Join(root, "gone.cu8")), AnalysisStatus: "running"}); err != nil {
		t.Fatal(err)
	}
	wav := filepath.Join(root, "Recordings", "2026-09-01", "20260901T120000.000Z-155000000-nfm.wav")
	if err := WriteMonoWAV(wav, make([]int16, 1600), 16000); err != nil {
		t.Fatal(err)
	}
	report, err := store.ReconcileMedia(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Recovered != 1 || report.Missing != 1 || report.Requeued != 1 {
		t.Fatalf("wrong report %+v", report)
	}
	reopened, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	again, err := reopened.ReconcileMedia(root)
	if err != nil || again.Recovered != 0 || reopened.Count() != 2 {
		t.Fatalf("recovery duplicated data: %+v %v", again, err)
	}
	e, _ := reopened.Get("missing")
	if len(e.MediaIssues) != 1 || e.AnalysisStatus != "unavailable" || e.IQPath != nil {
		t.Fatal("missing media hidden")
	}
	encoded, _ := os.ReadFile(filepath.Join(root, "Data", "media-recovery.json"))
	if !json.Valid(encoded) {
		t.Fatal("invalid report")
	}
}

func TestRuntimeCleanupReconcilesRemovedMedia(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	iq := filepath.Join(root, "IQ", "Pending", "old.cu8")
	if err := os.MkdirAll(filepath.Dir(iq), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iq, []byte("iq"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(TransmissionEvent{ID: "cleanup", IQPath: &iq, AnalysisStatus: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(iq); err != nil {
		t.Fatal(err)
	}
	if updated, err := store.ReconcileRemovedMedia(); err != nil || updated != 1 {
		t.Fatalf("runtime reconciliation failed: updated=%d err=%v", updated, err)
	}
	event, _ := store.Get("cleanup")
	if event.IQPath != nil || event.AnalysisStatus != "unavailable" || len(event.MediaIssues) != 1 || len(store.PendingAnalysis(0, "")) != 0 {
		t.Fatalf("removed media remained queued: %+v", event)
	}
}

func TestInterruptedJournalTailPreservesCommittedHistory(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(TransmissionEvent{ID: "committed"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTranscript("committed", "saved speech"); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(store.updatesPath(), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"id":"torn`)
	f.Close()
	reopened, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := reopened.Get("committed")
	if !ok || stringValue(e.Transcript) != "saved speech" {
		t.Fatal("committed update lost")
	}
	backups, _ := filepath.Glob(store.updatesPath() + ".interrupted-*")
	if len(backups) != 1 {
		t.Fatal("interrupted original not preserved")
	}
}

func TestRecoveryRemovesBannerEvidenceAndNoiseCaption(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(root, "Recordings", "test.wav")
	if err := WriteMonoWAV(audio, make([]int16, 8000), 8000); err != nil {
		t.Fatal(err)
	}
	transcript := "[sound of engine]"
	protocol := "P25"
	event := TransmissionEvent{ID: "bad-banner", AudioPath: &audio, Transcript: &transcript, ProtocolName: &protocol, AnalysisStatus: "complete", Analysis: &SignalIntelligence{SignalFamily: "P25", Confidence: .96}, DecoderMessages: []DecoderMessage{{DecoderID: "dsd-fme", Protocol: "P25", Summary: "Decoding AUTO P25, YSF, DSTAR, X2-TDMA, and DMR", RawText: "Decoding AUTO P25, YSF, DSTAR, X2-TDMA, and DMR"}}}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileMedia(root); err != nil {
		t.Fatal(err)
	}
	got, _ := store.Get(event.ID)
	if got.Transcript != nil || got.ProtocolName != nil || got.Analysis != nil || len(got.DecoderMessages) != 0 || got.AnalysisStatus != "pending" {
		t.Fatalf("invalid historical evidence survived: %+v", got)
	}
}

func TestOriginalArchiveProducesCorrectOfflineAudioWithoutChangingIQ(t *testing.T) {
	root := t.TempDir()
	const rate = 2_000_000
	data := syntheticFM(rate, .2, 100000, 1000, 5000)
	spec := CaptureSpec{SampleRateHz: rate, CenterFrequencyHz: 155000000}
	interval := CaptureInterval{ID: NewID(), ReceivedAt: time.Now(), SampleSeconds: .2, SampleBytes: len(data)}
	if err := writeCaptureInterval(root, &interval, spec, ComplexSigned8, data, true, 1<<30); err != nil {
		t.Fatal(err)
	}
	path, _, err := deriveStoredAudio(root, TransmissionEvent{ID: "audio", StartedAt: time.Now(), FrequencyHz: 155100000, BandwidthHz: 12500, Modulation: "NFM", IQPath: &interval.IQPath})
	if err != nil {
		t.Fatal(err)
	}
	audio, audioRate, err := readPCM16WAV(path)
	if err != nil {
		t.Fatal(err)
	}
	if frequency := zeroCrossingFrequency(audio[len(audio)/4:], audioRate); frequency < 900 || frequency > 1100 {
		t.Fatalf("offline archive audio frequency: %f", frequency)
	}
	if math.Abs(float64(len(audio))/float64(audioRate)-.2) > .002 {
		t.Fatal("audio duration does not match original interval")
	}
	original, _ := os.ReadFile(interval.IQPath)
	if !bytes.Equal(original, data) {
		t.Fatal("audio conversion changed original IQ")
	}
}

func TestMapperArchivesRepeatedHitsWithoutCooldown(t *testing.T) {
	r := mapperTestRuntime(t)
	spec := CaptureSpec{SampleRateHz: 250000, CenterFrequencyHz: 155000000}
	data := syntheticFM(250000, .1, 0, 1000, 5000)
	interval := CaptureInterval{ID: NewID(), ReceivedAt: time.Now(), SampleSeconds: .1, SampleBytes: len(data)}
	if err := writeCaptureInterval(r.dataDirectory, &interval, spec, ComplexSigned8, data, true, 1<<30); err != nil {
		t.Fatal(err)
	}
	run := &mapperRunContext{Config: MapperConfig{Mode: "discovery", CapturePolicy: "archive", AnalysisPolicy: "manual", NoiseMarginDB: 6}, Capture: interval}
	target := surveyTarget{FrequencyHz: 155000000, BandwidthHz: 12500, Mode: "nfm", Dwell: 100 * time.Millisecond}
	level := ChannelSpectrumLevel{SignalDB: -30, NoiseDB: -70, PeakDB: -25}
	before := r.Events.Count()
	for i := 0; i < 2; i++ {
		if !r.processSurveyTargetCapture(make(chan struct{}), ScanProfile{}, SDRDevice{ID: "test", Kind: "RTL-SDR"}, target, run, spec, data, ComplexSigned8, &level, true) {
			t.Fatal("capture failed")
		}
	}
	events := r.Events.Recent(2)
	if r.Events.Count() != before+2 || len(events) != 2 || stringValue(events[0].IQPath) != interval.IQPath || stringValue(events[1].IQPath) != interval.IQPath {
		t.Fatal("cooldown suppressed a hit or shared archive link was lost")
	}
}
