//go:build !linux

package app

func readPiPowerHat() PiPowerHatStatus { return PiPowerHatStatus{} }
