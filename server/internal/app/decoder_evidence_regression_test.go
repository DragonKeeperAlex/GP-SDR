package app

import "testing"

func TestEmptyDecoderMessagesCannotVerifyMapper(t *testing.T) {
	messages := []DecoderMessage{{Protocol: "ADS-B"}, {Summary: "startup"}, {Protocol: " ", RawText: "status"}, {Protocol: "ADS-B", Summary: "valid Mode S frame"}}
	valid := validDecoderMessages(messages)
	if len(valid) != 1 || valid[0].Summary != "valid Mode S frame" {
		t.Fatalf("invalid evidence accepted: %+v", valid)
	}
	for _, decoder := range []string{"dump1090", "rtl-433", "dsd-fme", "multimon-ng"} {
		if decoderLineIsEvidence(decoder, " \n\t") {
			t.Fatalf("%s accepted empty evidence", decoder)
		}
	}
}
