package app

import "time"

type PiPowerHatStatus struct {
	Available         bool      `json:"available"`
	InputVoltage      float64   `json:"inputVoltage"`
	InputCurrent      float64   `json:"inputCurrent"`
	InputPower        float64   `json:"inputPower"`
	OutputVoltage     float64   `json:"outputVoltage"`
	OutputCurrent     float64   `json:"outputCurrent"`
	OutputPower       float64   `json:"outputPower"`
	BatteryVoltage    float64   `json:"batteryVoltage"`
	BatteryCurrent    float64   `json:"batteryCurrent"`
	BatteryPower      float64   `json:"batteryPower"`
	BatteryPercentage float64   `json:"batteryPercentage"`
	PowerSource       string    `json:"powerSource"`
	InputPluggedIn    bool      `json:"inputPluggedIn"`
	Charging          bool      `json:"charging"`
	UpdatedAt         time.Time `json:"updatedAt,omitempty"`
	Error             string    `json:"error,omitempty"`
}
