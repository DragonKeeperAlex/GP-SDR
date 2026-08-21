package app

import "testing"

func TestParseTextDecoderOutputRequiresARealProtocolLine(t *testing.T) {
	messages := parseTextDecoderOutput("multimon-ng", "startup banner\nPOCSAG1200: Address: 123456 Function: 3 Alpha: TEST\n")
	if len(messages) != 1 || messages[0].Protocol != "POCSAG1200" || messages[0].Confidence < .9 {
		t.Fatalf("unexpected decoded messages: %#v", messages)
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
