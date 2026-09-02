// Package web contains the shared, offline GP-SDR interface.
package web

import "embed"

//go:embed *.html *.css *.js *.png *.svg reference
var Files embed.FS
