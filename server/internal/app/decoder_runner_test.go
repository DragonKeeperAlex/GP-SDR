package app

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTextDecoderOutputRequiresARealProtocolLine(t *testing.T) {
	messages := parseTextDecoderOutput("multimon-ng", "startup banner\nPOCSAG1200: Address: 123456 Function: 3 Alpha: TEST\n")
	if len(messages) != 1 || messages[0].Protocol != "POCSAG1200" || messages[0].Confidence < .9 {
		t.Fatalf("unexpected decoded messages: %#v", messages)
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
	for _, decoderID := range []string{"rtl-433", "dump1090", "acarsdec", "ais"} {
		_, err := runCandidateDecoder(context.Background(), decoderID, audio, 48_000, iqPath, 1090e6,
			CaptureSpec{CenterFrequencyHz: 1090125000, SampleRateHz: 1_000_000})
		if err != nil && (strings.Contains(strings.ToLower(err.Error()), "not installed") || strings.Contains(strings.ToLower(err.Error()), "not implemented")) {
			t.Fatalf("%s bridge unavailable: %v", decoderID, err)
		}
	}
}

func TestParseRTL433JSONOutput(t *testing.T) {
	messages := parseRTL433Output([]byte("noise\n{\"model\":\"Acurite-Tower\",\"id\":42,\"temperature_C\":21.2}\n"))
	if len(messages) != 1 || messages[0].Protocol != "ISM sensor" || messages[0].Summary != "Acurite-Tower · ID 42" {
		t.Fatalf("unexpected rtl_433 result: %#v", messages)
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
