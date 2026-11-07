package app

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// The notch, turned off, shows itself as the first song plays, and only
// then; its button keeps it on.
func TestNotchGreetsOnce(t *testing.T) {
	macOnly(t)
	a := newTestApp()
	n := &notch{a: a, w0: 200, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "宝船", Duration: 200000}})
	a.syncWindows()
	if n.hello {
		t.Fatal("greeted before the song played")
	}
	out := audio.NewPlayer(&silence{})
	out.SetVolume(0)
	a.pl.Attach(out)
	defer out.Close()
	a.syncWindows()
	if !n.hello || !a.settings.NotchGreeted {
		t.Fatal("did not greet as the song played")
	}

	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Turn On the Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !a.settings.Notch || n.hello {
		t.Fatalf("kept: notch %v, greeting %v", a.settings.Notch, n.hello)
	}

	// Turned off again, it does not greet the next song.
	a.settings.Notch = false
	a.pl.Cue([]music.Track{{ID: "8", Name: "下一首", Duration: 200000}})
	a.pl.Attach(out)
	a.syncWindows()
	if n.hello {
		t.Error("greeted twice")
	}
}

// The pointer resting on the greeting leaves it as it is, so its button can
// be clicked. It used to open the player, which hid the button, and leaving
// brought it back only to open again as the pointer came back to it.
func TestNotchGreetingKeptUnderPointer(t *testing.T) {
	macOnly(t)
	a := newTestApp()
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "宝船", Duration: 200000}})
	n.greet()
	if !n.hello {
		t.Fatal("did not greet")
	}

	defer func(f func() mygo.Point) { cursorPoint = f }(cursorPoint)
	cursorPoint = func() mygo.Point {
		return mygo.Point{X: int(n.screenX + n.winW/2 + 120), Y: int(n.screenY + n.h0 + 28)}
	}
	n.poll()
	n.enteredAt = n.lastInside.Add(-time.Second) // as if it had rested there
	n.poll()
	if n.open {
		t.Fatal("opened the player over the greeting")
	}

	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Turn On the Notch Player"); err != nil {
		t.Fatal(err)
	}
	if !a.settings.Notch || n.hello {
		t.Fatalf("kept: notch %v, greeting %v", a.settings.Notch, n.hello)
	}
}

// With MM_FRAMES set, saves the notch as it greets, growing and grown.
func TestFirstUseFrames(t *testing.T) {
	dir := os.Getenv("MM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	a := newTestApp()
	n := &notch{a: a, real: true, w0: 185, h0: 32, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	a.pl.Cue([]music.Track{{ID: "7", Name: "NAINAINAI", Artists: []music.Artist{{Name: "ATARASHII GAKKO!"}}, Duration: 193877}})
	main := ui.NewTester(a.view, 1200, 800) // for the app's colors
	main.Frame()
	tt := ui.NewTester(n.view, int(n.winW), int(n.winH))
	tt.Frame()
	n.greet()
	for _, ms := range []int{120, 600} {
		tt.Advance(time.Duration(ms) * time.Millisecond)
		tt.Frame()
		saveFrame(t, dir, fmt.Sprintf("notch-hello-%d.png", ms), tt.Image())
	}
}

// macOnly skips a test of the greeting, which only macOS shows: the notch
// is a Mac's.
func macOnly(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("the notch greets only on macOS")
	}
}

// Picking a display in the settings moves the notch to the top of it, and
// it goes back to the main display while that one is unplugged, and comes
// back when it's plugged in again, under another ID.
func TestNotchDisplayPicked(t *testing.T) {
	external := mygo.Display{ID: 3, Label: "DELL U2723QE", Bounds: mygo.Rectangle{Width: 2560, Height: 1440}, Primary: true}
	builtIn := mygo.Display{ID: 1, Label: "Built-in Retina Display", Bounds: mygo.Rectangle{X: 2560, Y: 400, Width: 1470, Height: 956}, Internal: true}
	ds := []mygo.Display{external, builtIn}
	defer func(f func() []mygo.Display) { displays = f }(displays)
	displays = func() []mygo.Display { return ds }

	a := newTestApp()
	a.settings.Notch = true
	n := &notch{a: a, winW: notchOpenW + 60, winH: 240}
	a.notch = n
	n.locate()
	if n.display.ID != external.ID {
		t.Fatalf("on display %d before any was picked, want the main one", n.display.ID)
	}

	a.settingsPage = "notch"
	tt := ui.NewTester(a.settingsView, 760, 900)
	tt.Frame()
	// The select, which shows what's picked.
	if err := tt.Click("Main Display"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(builtIn.Label); err != nil {
		t.Fatal(err)
	}
	on := func(d mygo.Display) bool {
		return n.display.ID == d.ID && n.screenX == float32(d.Bounds.X)+(float32(d.Bounds.Width)-n.winW)/2 && n.screenY == float32(d.Bounds.Y)
	}
	if !on(builtIn) {
		t.Fatalf("at %v,%v on display %d, want the top of the built-in one", n.screenX, n.screenY, n.display.ID)
	}

	ds = []mygo.Display{external}
	n.locate()
	if !on(external) {
		t.Fatalf("on display %d with the built-in one unplugged, want the main one", n.display.ID)
	}
	builtIn.ID = 5
	ds = []mygo.Display{external, builtIn}
	n.locate()
	if !on(builtIn) {
		t.Fatalf("on display %d with the built-in one plugged in again", n.display.ID)
	}
	if a.settings.NotchDisplay != builtIn.Label {
		t.Errorf("forgot the pick: %q", a.settings.NotchDisplay)
	}
}
