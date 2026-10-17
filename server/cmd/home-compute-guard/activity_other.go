//go:build !windows

package main

import "errors"

func nativeActivity() activity {
	return activity{Err: errors.New("Windows interactive session required")}
}
