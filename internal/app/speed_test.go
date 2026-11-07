package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// An episode's speed, chosen in the player bar, plays it and what follows at
// that speed, and is kept. A song shows its speed only while it isn't 1.
func TestEpisodeSpeed(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{{ID: "13", Name: "第一期", ShowID: "5", Duration: 600000}})
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

	a.pl.Cue([]music.Track{song("1", "老歌")})
	tt.Frame()
	if err := tt.Click("Playback Speed"); err != nil {
		t.Fatalf("a song played faster can't be slowed: %v", err)
	}
	if err := tt.ChooseMenuItem("1×"); err != nil {
		t.Fatalf("%v; the menu shows %q", err, tt.Menu())
	}
	tt.Frame()
	if a.pl.Speed() != 1 {
		t.Errorf("plays at %v", a.pl.Speed())
	}
	if _, ok := tt.Find("Playback Speed"); ok {
		t.Error("a song at 1 has a speed to choose")
	}
}
