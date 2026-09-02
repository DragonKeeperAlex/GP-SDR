module gpsdr.local/gpsdr/mobilebridge

go 1.26.0

require gpsdr.local/gpsdr v0.0.0

require (
	golang.org/x/mobile v0.0.0-20260821190718-4776eadac327 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
)

replace gpsdr.local/gpsdr => ..

tool golang.org/x/mobile/cmd/gobind
