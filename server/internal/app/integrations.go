package app

import "context"

type ComponentStatus struct {
	State string  `json:"state"`
	Note  string  `json:"note"`
	Path  *string `json:"path,omitempty"`
}

type IntegrationOverview struct {
	DSP            ComponentStatus      `json:"dsp"`
	SoapySDR       ComponentStatus      `json:"soapySDR"`
	P25            P25Status            `json:"p25"`
	Transcription  TranscriptionStatus  `json:"transcription"`
	RadioReference RadioReferenceStatus `json:"radioReference"`
}

func (r *Runtime) Integrations() IntegrationOverview {
	soapy := ComponentStatus{State: "setup", Note: "Install SoapySDR and a module for your receiver; the macOS stream helper is included."}
	if util, utilErr := findTool("SoapySDRUtil"); utilErr == nil {
		helper, helperErr := findTool("gpsdr-soapy")
		if helperErr != nil {
			helper, helperErr = findTool("signalharbor-soapy")
		}
		if helperErr == nil {
			soapy = ComponentStatus{State: "ready", Note: "SoapySDR discovery and streaming are ready.", Path: &helper}
		} else {
			soapy = ComponentStatus{State: "partial", Note: "SoapySDR is installed; build the GP-SDR stream helper for this platform to receive IQ.", Path: &util}
		}
	}
	r.mu.RLock()
	reference := r.radioReference
	r.mu.RUnlock()
	referenceStatus := RadioReferenceStatus{State: "setup", Endpoint: radioReferenceEndpoint, Note: "RadioReference is not configured."}
	if reference != nil {
		referenceStatus = reference.Status()
	}
	return IntegrationOverview{
		DSP:      ComponentStatus{State: "ready", Note: "Built-in AM, NFM, and WFM demodulation is ready."},
		SoapySDR: soapy, P25: r.P25Status(), Transcription: r.TranscriptionStatus(), RadioReference: referenceStatus,
	}
}

func (r *Runtime) RadioReferenceNearby(zipCode int, radiusMiles float64) (RadioReferenceNearbyResult, error) {
	r.mu.RLock()
	client := r.radioReference
	r.mu.RUnlock()
	if client == nil {
		return RadioReferenceNearbyResult{}, errInvalidRequest
	}
	return client.Nearby(context.Background(), zipCode, radiusMiles)
}
