//go:build !darwin

package app

func notchSize(display int64) (w, h float64, ok bool) { return 0, 0, false }

func raiseAboveMenuBar(handle uintptr) {}

func pinToTop(handle uintptr, display int64, w, h float64) {}

func menuBarHeight(display int64) float64 { return 0 }

func watchPointer(fn func()) (stop func(), ok bool) { return func() {}, false }
