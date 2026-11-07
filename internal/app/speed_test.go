package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// A speed chosen in the player bar, or over Now Playing, plays what's playing
// and what follows at it, and is kept. It used to be offered only for an
// episode in the player bar, so it couldn't be found while a song played.
func TestChooseSpeed(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "老歌")})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click("Playback Speed"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("1.5×"); err != nil {
		t.Fatalf("%v; the menu shows %q", err, tt.Menu())
	}
	if a.pl.Speed() != 1.5 || a.settings.Speed != 1.5 {
		t.Fatalf("plays at %v, keeps %v", a.pl.Speed(), a.settings.Speed)
	}

	a.nowPlaying = true
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	if err := tt.Click("Playback Speed"); err != nil {
		t.Fatalf("Now Playing has no speed: %v", err)
	}
	if err := tt.ChooseMenuItem("1×"); err != nil {
		t.Fatalf("%v; the menu shows %q", err, tt.Menu())
	}
	if a.pl.Speed() != 1 || a.settings.Speed != 1 {
		t.Errorf("plays at %v, keeps %v", a.pl.Speed(), a.settings.Speed)
	}
}
