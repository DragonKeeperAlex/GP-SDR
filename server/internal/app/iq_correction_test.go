package app

import "testing"

func TestDetectOffsetBinaryHackRFSamples(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(132 + (i % 5))
	}
	if got := DetectSampleFormat(data, ComplexSigned8); got != ComplexUnsigned8 {
		t.Fatalf("expected unsigned offset-binary detection, got %s", got)
	}
}

func TestIQCorrectionRemovesDC(t *testing.T) {
	data := []byte{20, 10, 20, 10, 20, 10, 20, 10}
	ApplyIQCorrection(data, ComplexSigned8, true, 1, 0, false)
	for _, sample := range data {
		if int8(sample) != 0 {
			t.Fatalf("DC remains: %v", data)
		}
	}
}

func TestIQCorrectionSwapsChannels(t *testing.T) {
	data := []byte{10, 20}
	ApplyIQCorrection(data, ComplexSigned8, false, 1, 0, true)
	if int8(data[0]) != 20 || int8(data[1]) != 10 {
		t.Fatalf("not swapped: %v", data)
	}
}
