package app

import (
	"fmt"
	"math"
)

// P25ReceiverSettings is an optional profile override for one physical receiver.
// Nil values retain existing shared settings or saved calibration.
type P25ReceiverSettings struct {
	SampleRateHz  int      `json:"sampleRateHz,omitempty"`
	GainDB        *float64 `json:"gainDB,omitempty"`
	LNAGainDB     *int     `json:"lnaGainDB,omitempty"`
	VGAGainDB     *int     `json:"vgaGainDB,omitempty"`
	AmpEnabled    *bool    `json:"ampEnabled,omitempty"`
	PPMCorrection *int     `json:"ppmCorrection,omitempty"`
}

func validateP25ReceiverSettings(settings P25ReceiverSettings) error {
	_, rtlRate := rtlSDRSampleRateName(settings.SampleRateHz)
	if settings.SampleRateHz != 0 && !supportedUserSampleRate(settings.SampleRateHz) && !rtlRate {
		return fmt.Errorf("unsupported receiver rate %d", settings.SampleRateHz)
	}
	if gain := settings.GainDB; gain != nil && (math.IsNaN(*gain) || math.IsInf(*gain, 0) || *gain < 0 || *gain > 73) {
		return fmt.Errorf("receiver gain must be 0 to 73 dB")
	}
	if gain := settings.LNAGainDB; gain != nil && (*gain < 0 || *gain > 40 || *gain%8 != 0) {
		return fmt.Errorf("HackRF LNA must be 0 to 40 dB in steps of 8")
	}
	if gain := settings.VGAGainDB; gain != nil && (*gain < 0 || *gain > 62 || *gain%2 != 0) {
		return fmt.Errorf("HackRF VGA must be 0 to 62 dB in steps of 2")
	}
	if ppm := settings.PPMCorrection; ppm != nil && (*ppm < -200 || *ppm > 200) {
		return fmt.Errorf("receiver correction must be -200 to 200 ppm")
	}
	return nil
}

func applyP25ReceiverSettings(config *op25Device, device SDRDevice, settings P25ReceiverSettings) error {
	if err := validateP25ReceiverSettings(settings); err != nil {
		return err
	}
	if rate := settings.SampleRateHz; rate > 0 {
		valid := false
		switch device.Kind {
		case "HackRF":
			valid = isHackRFSampleRate(rate)
		case "RTL-SDR":
			_, valid = rtlSDRSampleRateName(rate)
		case "PlutoSDR":
			valid = rate == 1_000_000 || rate == 2_000_000 || rate == 2_400_000 || rate == 5_000_000 || rate == 10_000_000
		}
		if !valid || device.SampleRateLimit != nil && float64(rate) > *device.SampleRateLimit {
			return fmt.Errorf("%s does not support P25 capture at %d samples/s", device.Name, rate)
		}
		config.Rate = rate
	}
	if device.Kind == "HackRF" {
		if settings.GainDB != nil {
			return fmt.Errorf("use separate HackRF LNA and VGA gain controls")
		}
		lna, vga, amp := 8, 0, 0
		// Preserve shared profile defaults when only one receiver field changes.
		_, _ = fmt.Sscanf(config.Gains, "LNA:%d,VGA:%d,AMP:%d", &lna, &vga, &amp)
		if settings.LNAGainDB != nil {
			lna = *settings.LNAGainDB
		}
		if settings.VGAGainDB != nil {
			vga = *settings.VGAGainDB
		}
		if settings.AmpEnabled != nil {
			amp = 0
			if *settings.AmpEnabled {
				amp = 14
			}
		}
		config.Gains = fmt.Sprintf("LNA:%d,VGA:%d,AMP:%d", lna, vga, amp)
	} else {
		if settings.LNAGainDB != nil || settings.VGAGainDB != nil || settings.AmpEnabled != nil {
			return fmt.Errorf("HackRF front-end controls cannot be applied to %s", device.Kind)
		}
		if gain := settings.GainDB; gain != nil {
			switch device.Kind {
			case "RTL-SDR":
				if *gain > 49 {
					return fmt.Errorf("RTL-SDR gain must be 0 to 49 dB")
				}
				config.Gains = fmt.Sprintf("LNA:%g", *gain)
			case "PlutoSDR":
				config.Gains = fmt.Sprintf("PGA:%g", *gain)
			default:
				return fmt.Errorf("P25 gain override unsupported for %s", device.Kind)
			}
		}
	}
	if settings.PPMCorrection != nil {
		config.PPM = float64(*settings.PPMCorrection)
	}
	return nil
}
