package app

import "testing"

func TestIQInputWarningDetectsPinnedSignedQuadraturePath(t *testing.T) {
	data := make([]byte, 4096)
	for index := 0; index < len(data); index += 2 {
		data[index] = 12
		data[index+1] = 129 // signed int8 -127
	}
	if got := iqInputWarning(data, ComplexSigned8); got == "" {
		t.Fatal("expected pinned Q input warning")
	}
}

func TestIQInputWarningAllowsNormalNoise(t *testing.T) {
	data := deterministicIQNoise(4096, .2)
	if got := iqInputWarning(data, ComplexSigned8); got != "" {
		t.Fatalf("normal complex noise should not be flagged: %s", got)
	}
}
