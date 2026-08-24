package app

import (
	"math"
	"strings"
)

// DecoderCandidate describes what a decoder may be able to prove. A band or
// frequency match is deliberately only a candidate; GP-SDR must not turn RF
// energy into a protocol claim without valid decoder output.
type DecoderCandidate struct {
	DecoderID string
	Protocol  string
	Label     string
	Mode      string
	Reason    string
	Exact     bool
}

func decoderScanProfiles() []ScanProfile {
	profiles := []ScanProfile{
		decoderRangeProfile("decoder-dmr", "DMR Conventional", "DMR Tier I/II channels with two-slot voice and metadata decoding", "DMR", "dmr",
			decoderRange("VHF DMR 136–155", 136e6, 155e6, 12_500, "dmr", "dmr"),
			decoderRange("VHF DMR 155–174", 155e6, 174e6, 12_500, "dmr", "dmr"),
			decoderRange("UHF DMR 400–420", 400e6, 420e6, 12_500, "dmr", "dmr"),
			decoderRange("UHF DMR 420–440", 420e6, 440e6, 12_500, "dmr", "dmr"),
			decoderRange("UHF DMR 440–460", 440e6, 460e6, 12_500, "dmr", "dmr"),
			decoderRange("UHF DMR 460–480", 460e6, 480e6, 12_500, "dmr", "dmr"),
			decoderRange("900 MHz DMR 935–941", 935e6, 941e6, 12_500, "dmr", "dmr")),
		decoderRangeProfile("decoder-dsd-fme", "Digital Voice Discovery", "Conventional digital voice candidates for DSD-FME", "DSD-FME", "dsd-fme",
			decoderRange("VHF digital voice 136–155", 136e6, 155e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("VHF digital voice 155–174", 155e6, 174e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("UHF digital voice 400–420", 400e6, 420e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("UHF digital voice 420–440", 420e6, 440e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("UHF digital voice 440–460", 440e6, 460e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("UHF digital voice 460–480", 460e6, 480e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("700 MHz digital voice candidates", 769e6, 775e6, 12_500, "nfm", "dsd-fme"),
			decoderRange("800 MHz digital voice candidates", 851e6, 869e6, 12_500, "nfm", "dsd-fme")),
		decoderChannelProfile("decoder-rtl-433", "ISM Sensors", "Common rtl_433 sensor, weather, and TPMS centers", "rtl_433", "rtl-433",
			decoderChannel("315 MHz ISM", 315.000, 250_000, "nfm", "rtl-433"),
			decoderChannel("345 MHz ISM", 345.000, 250_000, "nfm", "rtl-433"),
			decoderChannel("433.92 MHz ISM", 433.920, 250_000, "nfm", "rtl-433"),
			decoderChannel("868 MHz SRD", 868.000, 250_000, "nfm", "rtl-433"),
			decoderChannel("915 MHz ISM", 915.000, 250_000, "nfm", "rtl-433")),
		decoderChannelProfile("decoder-dump1090", "ADS-B · 1090 MHz", "Aircraft ADS-B and Mode S target", "dump1090", "dump1090",
			decoderChannel("ADS-B / Mode S", 1090.000, 2_000_000, "nfm", "dump1090")),
		decoderRangeProfile("decoder-multimon-ng", "Paging & Signaling Discovery", "Common US paging and signaling candidate bands", "multimon-ng", "multimon-ng",
			decoderRange("VHF paging candidates", 152e6, 153e6, 12_500, "nfm", "multimon-ng"),
			decoderRange("UHF paging candidates", 454e6, 460e6, 12_500, "nfm", "multimon-ng"),
			decoderRange("900 MHz paging candidates", 929e6, 932e6, 12_500, "nfm", "multimon-ng")),
		decoderChannelProfile("decoder-acarsdec", "ACARS · North America", "Common North American VHF ACARS channels", "ACARS", "acarsdec",
			decoderChannel("ACARS 130.025", 130.025, 25_000, "am", "acarsdec"),
			decoderChannel("ACARS 131.550", 131.550, 25_000, "am", "acarsdec"),
			decoderChannel("ACARS 131.725", 131.725, 25_000, "am", "acarsdec")),
		decoderChannelProfile("decoder-ais", "Marine AIS", "AIS 1 and AIS 2 vessel data channels", "AIS", "ais",
			decoderChannel("AIS 1 · 87B", 161.975, 25_000, "nfm", "ais"),
			decoderChannel("AIS 2 · 88B", 162.025, 25_000, "nfm", "ais")),
	}
	return profiles
}

func decoderChannelProfile(id, name, summary, target, _ string, channels ...ChannelDefinition) ScanProfile {
	profile := fixedChannelProfile(id, name, summary, target)
	profile.Channels = channels
	return profile
}

func decoderRangeProfile(id, name, summary, target, _ string, ranges ...ScanRange) ScanProfile {
	profile := rangeProfile(id, name, summary, target, ranges...)
	return profile
}

func decoderChannel(name string, mhz, bandwidth float64, mode, decoder string) ChannelDefinition {
	item := channel(name, mhz, bandwidth, mode)
	item.Decoder = ptr(decoder)
	return item
}

func decoderRange(name string, startHz, endHz, stepHz float64, mode, decoder string) ScanRange {
	return ScanRange{ID: NewID(), Name: name, StartHz: startHz, EndHz: endHz, StepHz: stepHz,
		DwellMilliseconds: decoderDwellMilliseconds(decoder), PreferredMode: mode, Decoder: ptr(decoder), Enabled: true}
}

func decoderDwellMilliseconds(decoder string) int {
	if canonicalDecoderID(decoder) == "dsd-fme" {
		return 2500
	}
	return 300
}

func decoderCandidate(frequencyHz float64, explicitDecoder string) (DecoderCandidate, bool) {
	requestedDecoder := strings.ToLower(strings.TrimSpace(explicitDecoder))
	MHz := frequencyHz / 1e6
	if requestedDecoder != "" {
		return candidateForDecoder(requestedDecoder, MHz, true), true
	}
	switch {
	case nearMHz(MHz, 1090, 1):
		return candidateForDecoder("dump1090", MHz, true), true
	case nearAnyMHz(MHz, .15, 315, 345, 433.92, 868, 915):
		return candidateForDecoder("rtl-433", MHz, true), true
	case nearAnyMHz(MHz, .015, 130.025, 131.550, 131.725):
		return candidateForDecoder("acarsdec", MHz, true), true
	case nearAnyMHz(MHz, .015, 161.975, 162.025):
		return candidateForDecoder("ais", MHz, true), true
	case (MHz >= 152 && MHz <= 153) || (MHz >= 454 && MHz <= 460) || (MHz >= 929 && MHz <= 932):
		return candidateForDecoder("multimon-ng", MHz, false), true
	}
	return DecoderCandidate{}, false
}

func candidateForDecoder(decoder string, MHz float64, exact bool) DecoderCandidate {
	requested := strings.ToLower(strings.TrimSpace(decoder))
	switch canonicalDecoderID(decoder) {
	case "dsd-fme":
		if protocol := digitalVoiceProtocol(requested); protocol != "" && protocol != "Digital voice" {
			return DecoderCandidate{DecoderID: requested, Protocol: protocol + " candidate", Label: protocol + " activity", Mode: "DIGITAL", Reason: "RF activity assigned to the " + protocol + " decoder", Exact: exact}
		}
		return DecoderCandidate{DecoderID: "dsd-fme", Protocol: "Digital voice candidate", Label: "Digital voice activity", Mode: "DIGITAL", Reason: "RF activity in a configured DSD-FME scan target", Exact: exact}
	case "rtl-433":
		return DecoderCandidate{DecoderID: "rtl-433", Protocol: "ISM sensor candidate", Label: "ISM sensor activity", Mode: "DIGITAL", Reason: "RF activity near a common rtl_433 center frequency", Exact: exact}
	case "dump1090":
		return DecoderCandidate{DecoderID: "dump1090", Protocol: "ADS-B / Mode S candidate", Label: "Aircraft transponder activity", Mode: "DIGITAL", Reason: "RF activity near 1090 MHz", Exact: exact}
	case "multimon-ng":
		return DecoderCandidate{DecoderID: "multimon-ng", Protocol: "Paging / signaling candidate", Label: "Paging or signaling activity", Mode: "DIGITAL", Reason: "RF activity in a configured paging/signaling band", Exact: exact}
	case "acarsdec":
		return DecoderCandidate{DecoderID: "acarsdec", Protocol: "ACARS candidate", Label: "Aircraft datalink activity", Mode: "AM", Reason: "RF activity on a common North American ACARS channel", Exact: exact}
	case "ais":
		return DecoderCandidate{DecoderID: "ais", Protocol: "AIS candidate", Label: "Marine vessel data activity", Mode: "DIGITAL", Reason: "RF activity on AIS 1 or AIS 2", Exact: exact}
	}
	return DecoderCandidate{DecoderID: decoder, Protocol: strings.TrimSpace(decoder) + " candidate", Label: "Decoder candidate", Mode: "DIGITAL", Reason: "RF activity on a decoder-assigned target", Exact: exact}
}

func canonicalDecoderID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "rtl_433", "rtl433":
		return "rtl-433"
	case "ads-b", "adsb", "mode s", "ads-b / mode s":
		return "dump1090"
	case "ais-catcher":
		return "ais"
	case "acars":
		return "acarsdec"
	case "pocsag", "flex", "mdc1200", "dtmf":
		return "multimon-ng"
	case "dmr", "nxdn", "d-star", "ysf", "m17":
		return "dsd-fme"
	}
	return value
}

func digitalVoiceProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dmr":
		return "DMR"
	case "p25", "p25 phase 1", "p25 phase 2":
		return "P25"
	case "nxdn", "nxdn48", "nxdn96":
		return "NXDN"
	case "d-star", "dstar":
		return "D-STAR"
	case "ysf":
		return "YSF"
	case "m17":
		return "M17"
	case "digital", "dsd-fme", "auto-digital":
		return "Digital voice"
	}
	return ""
}

func decoderForMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "p25" || mode == "p25 phase 1" || mode == "p25 phase 2" {
		return "dsd-fme"
	}
	if digitalVoiceProtocol(mode) != "" {
		return mode
	}
	switch mode {
	case "adsb", "ads-b", "mode-s", "mode s":
		return "dump1090"
	case "rtl-433", "sensors":
		return "rtl-433"
	case "pocsag", "flex", "signaling":
		return "multimon-ng"
	case "acars":
		return "acarsdec"
	case "ais":
		return "ais"
	}
	return ""
}

func demodulationModeForDecoder(mode, decoder string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if canonicalDecoderID(decoder) == "dsd-fme" || digitalVoiceProtocol(mode) != "" {
		return "nfm"
	}
	if mode == "acars" {
		return "am"
	}
	if mode == "auto" || mode == "" || mode == "am" || mode == "nfm" || mode == "wfm" || mode == "fm" {
		return mode
	}
	return "nfm"
}

func decoderBandwidthHz(decoder string, fallback float64) float64 {
	switch canonicalDecoderID(decoder) {
	case "dump1090":
		return 2_000_000
	case "rtl-433":
		return math.Max(fallback, 250_000)
	case "ais", "acarsdec":
		return math.Max(fallback, 25_000)
	default:
		return math.Max(fallback, 12_500)
	}
}

func nearMHz(value, target, tolerance float64) bool { return math.Abs(value-target) <= tolerance }

func nearAnyMHz(value, tolerance float64, targets ...float64) bool {
	for _, target := range targets {
		if nearMHz(value, target, tolerance) {
			return true
		}
	}
	return false
}
