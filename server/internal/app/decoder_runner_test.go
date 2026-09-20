package app

import (
	"bytes"
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransientDecoderIQIsNotRetained(t *testing.T) {
	path, cleanup, err := writeTransientDecoderIQ([]byte{0, 1, 2, 3}, ComplexSigned8)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(path) != ".cs8" {
		t.Fatalf("unexpected transient extension %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, []byte{0, 1, 2, 3}) {
		t.Fatalf("transient IQ mismatch: %v %v", data, err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("transient IQ survived cleanup: %v", err)
	}
}

func TestDecoderNeedsIQ(t *testing.T) {
	for _, decoderID := range []string{"rtl-433", "dump1090", "dump978", "ais"} {
		if !decoderNeedsIQ(decoderID) {
			t.Fatalf("%s should require IQ", decoderID)
		}
	}
	for _, decoderID := range []string{"multimon-ng", "dsd-fme", "acarsdec"} {
		if decoderNeedsIQ(decoderID) {
			t.Fatalf("%s should not require IQ", decoderID)
		}
	}
}

func TestParseTextDecoderOutputRequiresARealProtocolLine(t *testing.T) {
	messages := parseTextDecoderOutput("multimon-ng", "startup banner\nPOCSAG1200: Address: 123456 Function: 3 Alpha: TEST\n")
	if len(messages) != 1 || messages[0].Protocol != "POCSAG1200" || messages[0].Confidence < .9 {
		t.Fatalf("unexpected decoded messages: %#v", messages)
	}
}

func TestParseDMRMetadata(t *testing.T) {
	messages := parseTextDecoderOutput("dsd-fme", "2026-08-24 DMR Voice Slot 2 CC: 7 TGT: 1949001 SRC: 50\n")
	if len(messages) != 1 {
		t.Fatalf("expected one DMR message, got %#v", messages)
	}
	message := messages[0]
	if message.Protocol != "DMR" || message.TimeSlot != 2 || message.ColorCode != 7 || message.Talkgroup != 1949001 || message.SourceID != 50 {
		t.Fatalf("unexpected DMR metadata: %#v", message)
	}
}

func TestDSDStatusBannerIsNotAReceivedFrame(t *testing.T) {
	messages := parseTextDecoderOutput("dsd-fme", "Decoding AUTO P25, YSF, DSTAR, X2-TDMA, and DMR\n")
	if len(messages) != 0 {
		t.Fatalf("status banner was accepted as RF evidence: %#v", messages)
	}
}

func TestDSDModeFlags(t *testing.T) {
	for mode, expected := range map[string]string{"dmr": "-fs", "p25": "-ft", "p25 phase 1": "-f1", "nxdn48": "-fi", "nxdn": "-fn", "d-star": "-fd", "ysf": "-fy", "m17": "-fz", "digital": "-fa"} {
		if actual := dsdModeFlag(mode); actual != expected {
			t.Fatalf("%s: expected %s, got %s", mode, expected, actual)
		}
	}
}

func TestReadPCM16WAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "voice.wav")
	original := []int16{100, -200, 300, -400}
	if err := WriteMonoWAV(path, original, 8_000); err != nil {
		t.Fatal(err)
	}
	decoded, rate, err := readPCM16WAV(path)
	if err != nil || rate != 8_000 || len(decoded) != len(original) {
		t.Fatalf("unexpected WAV result: rate=%d samples=%d err=%v", rate, len(decoded), err)
	}
}

func TestInstalledOptionalDecoderBridgesSmoke(t *testing.T) {
	if os.Getenv("GPSDR_DECODER_SMOKE") != "1" {
		t.Skip("set GPSDR_DECODER_SMOKE=1 to exercise installed external decoders")
	}
	directory := t.TempDir()
	iqPath := filepath.Join(directory, "silence.cs8")
	if err := os.WriteFile(iqPath, make([]byte, 200_000), 0o600); err != nil {
		t.Fatal(err)
	}
	audio := make([]int16, 24_000)
	for _, decoderID := range []string{"dmr", "rtl-433", "dump1090", "multimon-ng", "acarsdec", "ais"} {
		messages, err := runCandidateDecoder(context.Background(), decoderID, audio, 48_000, iqPath, 1090e6,
			CaptureSpec{CenterFrequencyHz: 1090125000, SampleRateHz: 1_000_000})
		if err != nil && (strings.Contains(strings.ToLower(err.Error()), "not installed") || strings.Contains(strings.ToLower(err.Error()), "not implemented")) {
			t.Fatalf("%s bridge unavailable: %v", decoderID, err)
		}
		if decoderID == "dmr" && len(messages) != 0 {
			t.Fatalf("DSD-FME treated silence/status output as received frames: %#v", messages)
		}
	}
}

func TestParseRTL433JSONOutput(t *testing.T) {
	messages := parseRTL433Output([]byte("noise\n{\"model\":\"Acurite-Tower\",\"id\":42,\"temperature_C\":21.2}\n"))
	if len(messages) != 1 || messages[0].Protocol != "ISM sensor" || messages[0].Summary != "Acurite-Tower · ID 42" {
		t.Fatalf("unexpected rtl_433 result: %#v", messages)
	}
}

func TestParseRTL433IgnoresNullIdentityFields(t *testing.T) {
	messages := parseRTL433Output([]byte(`{"model":null,"id":null,"temperature_C":21.2}` + "\n"))
	if len(messages) != 1 || messages[0].Summary != "Decoded ISM sensor frame" {
		t.Fatalf("null identity fields leaked into sensor summary: %#v", messages)
	}
}

func TestResamplePCMProducesExpectedDuration(t *testing.T) {
	input := make([]int16, 16_000)
	if output := resamplePCM(input, 16_000, 48_000); len(output) != 48_000 {
		t.Fatalf("expected one second at 48 kHz, got %d samples", len(output))
	}
}

func TestParseDump1090Frames(t *testing.T) {
	messages := parseDump1090Output([]byte("banner\n*8D40621D58C382D690C8AC2863A7;\n"))
	if len(messages) != 1 || messages[0].Protocol != "ADS-B / Mode S" || messages[0].Summary != "Mode S frame · ICAO 40621D" {
		t.Fatalf("unexpected dump1090 messages: %#v", messages)
	}
}

func TestParseDump978Frames(t *testing.T) {
	messages := parseDump978Output([]byte("uat startup\n+8D40621D58C382D690C8AC2863A7;\n"))
	if len(messages) != 1 || messages[0].Protocol != "UAT / ADS-B" {
		t.Fatalf("unexpected UAT messages: %#v", messages)
	}
}

func TestParseDirewolfFrames(t *testing.T) {
	messages := parseDirewolfOutput([]byte("Dire Wolf version 1.7\nN0CALL>APRS,WIDE1-1:!3745.12N/12225.45W-Test\n"))
	if len(messages) != 1 || messages[0].Protocol != "APRS / AX.25" || len(messages[0].Callsigns) == 0 {
		t.Fatalf("unexpected Dire Wolf messages: %#v", messages)
	}
}

func TestDump1090FiniteNoFrameExitIsNotDecoderFailure(t *testing.T) {
	output := []byte("dump1090-fa starting up\nWaiting for receive thread termination\nAbnormal exit.\n")
	if !dump1090ReachedEOF(output) {
		t.Fatal("expected finite no-frame dump1090 output to be recognized")
	}
	if dump1090ReachedEOF([]byte("ERROR: unable to open input\nWaiting for receive thread termination\n")) {
		t.Fatal("input errors must remain visible")
	}
}

func TestParseACARSAndAISOutput(t *testing.T) {
	acars := parseACARSOutput([]byte("Starting\n#2 (L: -5 E:0) .N842UA UA123 H1 TEST\nexiting\n"))
	if len(acars) != 1 || acars[0].Protocol != "ACARS" {
		t.Fatalf("unexpected ACARS result: %#v", acars)
	}
	ais := parseAISOutput([]byte("{\"mmsi\":367123456,\"shipname\":\"TEST VESSEL\",\"callsign\":\"WDF1234\"}\n"))
	if len(ais) != 1 || ais[0].Protocol != "AIS" || len(ais[0].Callsigns) != 1 {
		t.Fatalf("unexpected AIS result: %#v", ais)
	}
}

func TestParseAISAlternateFieldNames(t *testing.T) {
	ais := parseAISOutput([]byte(`{"MMSI":367123456,"VesselName":"TEST VESSEL","callSign":"WDF1234"}` + "\n"))
	if len(ais) != 1 || !strings.Contains(ais[0].Summary, "MMSI 367123456") || !strings.Contains(ais[0].Summary, "TEST VESSEL") || len(ais[0].Callsigns) != 1 {
		t.Fatalf("alternate AIS field names were not extracted: %#v", ais)
	}
}

func TestPrepareUC8DecoderIQCentersAndResamples(t *testing.T) {
	const sourceRate = 1_000_000
	const offset = -125_000.0
	samples := make([]byte, sourceRate/100*2)
	for index := 0; index < len(samples)/2; index++ {
		phase := 2 * math.Pi * offset * float64(index) / sourceRate
		samples[index*2] = byte(int8(math.Round(80 * math.Cos(phase))))
		samples[index*2+1] = byte(int8(math.Round(80 * math.Sin(phase))))
	}
	input := filepath.Join(t.TempDir(), "capture.cs8")
	if err := os.WriteFile(input, samples, 0o600); err != nil {
		t.Fatal(err)
	}
	path, cleanup, err := prepareUC8DecoderIQ(input, 1090e6, CaptureSpec{CenterFrequencyHz: 1090125000, SampleRateHz: sourceRate}, 2_400_000)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != (len(samples)/2)*2_400_000/sourceRate*2 {
		t.Fatalf("unexpected prepared length: %d", len(output))
	}
	qMagnitude := 0.0
	for index := 0; index < len(output)/2; index++ {
		qMagnitude += math.Abs(float64(output[index*2+1]) - 128)
	}
	if qMagnitude/float64(len(output)/2) > 4 {
		t.Fatalf("frequency shift did not center the carrier: mean Q %.2f", qMagnitude/float64(len(output)/2))
	}
}
