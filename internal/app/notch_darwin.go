//go:build darwin

package app

import "github.com/ebitengine/purego/objc"

type nsPoint struct{ X, Y float64 }
type nsSize struct{ W, H float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}
type nsEdgeInsets struct{ Top, Left, Bottom, Right float64 }

// notchSize returns the size of the camera housing of the display with the
// ID in points, or false if it has none.
func notchSize(display int64) (w, h float64, ok bool) {
	s := screenOf(display)
	if s == 0 || !objc.Send[bool](s, objc.RegisterName("respondsToSelector:"), objc.RegisterName("safeAreaInsets")) {
		return 0, 0, false
	}
	insets := objc.Send[nsEdgeInsets](s, objc.RegisterName("safeAreaInsets"))
	if insets.Top <= 0 {
		return 0, 0, false
	}
	frame := objc.Send[nsRect](s, objc.RegisterName("frame"))
	left := objc.Send[nsRect](s, objc.RegisterName("auxiliaryTopLeftArea"))
	right := objc.Send[nsRect](s, objc.RegisterName("auxiliaryTopRightArea"))
	return frame.Size.W - left.Size.W - right.Size.W, insets.Top, true
}

// menuBarHeight returns the height of the menu bar of the display with the
// ID, or 0 if it has none.
func menuBarHeight(display int64) float64 {
	s := screenOf(display)
	if s == 0 {
		return 0
	}
	f := objc.Send[nsRect](s, objc.RegisterName("frame"))
	v := objc.Send[nsRect](s, objc.RegisterName("visibleFrame"))
	return f.Origin.Y + f.Size.H - (v.Origin.Y + v.Size.H)
}

// raiseAboveMenuBar puts a window above the menu bar, where the notch is,
// on every Space and over apps in full screen.
func raiseAboveMenuBar(handle uintptr) {
	win := objc.ID(handle)
	// MyGo makes every window titled, its title bar hidden, and AppKit
	// keeps titled windows below the menu bar: make this one borderless,
	// keeping its panel's style, which never activates the app.
	const nonactivatingPanel, fullSizeContentView = 1 << 7, 1 << 15
	win.Send(objc.RegisterName("setStyleMask:"), uint(nonactivatingPanel|fullSizeContentView))
	const mainMenuLevel = 24 // NSMainMenuWindowLevel
	win.Send(objc.RegisterName("setLevel:"), mainMenuLevel+3)
	// Joins all Spaces, stays as Spaces switch, full-screen auxiliary,
	// out of the window cycle.
	win.Send(objc.RegisterName("setCollectionBehavior:"), uint(1|1<<4|1<<8|1<<6))
	// Hover while another app is active: MyGo tracks the pointer only in
	// the active app, and the notch is always over another one.
	if view := surfaceView(win.Send(objc.RegisterName("contentView"))); view != 0 {
		const options = 0x01 | 0x02 | 0x04 | 0x80 | 0x200 // entered/exited, moved, cursor, always, visible rect
		area := objc.ID(objc.GetClass("NSTrackingArea")).Send(objc.RegisterName("alloc"))
		area = area.Send(objc.RegisterName("initWithRect:options:owner:userInfo:"), nsRect{}, uint(options), view, objc.ID(0))
		view.Send(objc.RegisterName("addTrackingArea:"), area)
	}
}

// surfaceView finds the view MyGo draws native UI in.
func surfaceView(v objc.ID) objc.ID {
	if v == 0 {
		return 0
	}
	if objc.Send[bool](v, objc.RegisterName("isKindOfClass:"), objc.GetClass("MyGoSurfaceView")) {
		return v
	}
	subs := v.Send(objc.RegisterName("subviews"))
	for i := range objc.Send[uint](subs, objc.RegisterName("count")) {
		if found := surfaceView(subs.Send(objc.RegisterName("objectAtIndex:"), i)); found != 0 {
			return found
		}
	}
	return 0
}

// pinToTop moves a window of w×h points to the top middle of the display
// with the ID, over its menu bar, where AppKit would not otherwise let a
// window go.
func pinToTop(handle uintptr, display int64, w, h float64) {
	screen := screenOf(display)
	if screen == 0 {
		return
	}
	f := objc.Send[nsRect](screen, objc.RegisterName("frame"))
	r := nsRect{nsPoint{f.Origin.X + (f.Size.W-w)/2, f.Origin.Y + f.Size.H - h}, nsSize{w, h}}
	objc.ID(handle).Send(objc.RegisterName("setFrame:display:"), r, true)
}

// screenOf returns the screen of the display with the ID, or, when it is
// gone, the one with the menu bar: the first.
func screenOf(display int64) objc.ID {
	screens := objc.ID(objc.GetClass("NSScreen")).Send(objc.RegisterName("screens"))
	count := objc.Send[uint](screens, objc.RegisterName("count"))
	if count == 0 {
		return 0
	}
	key := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), "NSScreenNumber")
	for i := range count {
		s := screens.Send(objc.RegisterName("objectAtIndex:"), i)
		num := s.Send(objc.RegisterName("deviceDescription")).Send(objc.RegisterName("objectForKey:"), key)
		if num != 0 && int64(objc.Send[uint32](num, objc.RegisterName("unsignedIntValue"))) == display {
			return s
		}
	}
	return screens.Send(objc.RegisterName("objectAtIndex:"), uint(0))
}

// watchPointer calls fn on the main thread as the pointer moves, over this
// app's windows or another's, until stop is called.
func watchPointer(fn func()) (stop func(), ok bool) {
	// mouse moved, and left, right and other mouse dragged
	const mask = uint64(1<<5 | 1<<6 | 1<<7 | 1<<27)
	events := objc.ID(objc.GetClass("NSEvent"))
	global := objc.NewBlock(func(_ objc.Block, _ objc.ID) { fn() })
	local := objc.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID { fn(); return ev })
	g := events.Send(objc.RegisterName("addGlobalMonitorForEventsMatchingMask:handler:"), mask, global)
	l := events.Send(objc.RegisterName("addLocalMonitorForEventsMatchingMask:handler:"), mask, local)
	g.Send(objc.RegisterName("retain"))
	l.Send(objc.RegisterName("retain"))
	return func() {
		for _, m := range []objc.ID{g, l} {
			if m != 0 {
				events.Send(objc.RegisterName("removeMonitor:"), m)
				m.Send(objc.RegisterName("release"))
			}
		}
		global.Release()
		local.Release()
	}, g != 0 && l != 0
}
