// SPDX-License-Identifier: GPL-3.0-only
//go:build windows

package main

import "os"

func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
