package app

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// Each artist's name opens their page: from Now Playing, which closes, and
// from a song's row, which doesn't play the song as the row's click would.
func TestArtistNameOpensTheirPage(t *testing.T) {
	a := newTestApp()
	duet := music.Track{ID: "1", Name: "广寒宫", Artists: []music.Artist{{ID: "11", Name: "甲"}, {ID: "12", Name: "乙"}}, Duration: 200000}
	a.pl.Cue([]music.Track{duet})
	a.pl.SetLyrics(nil)
	a.artists["12"] = &artistData{page: &music.ArtistPage{
		Artist: music.Artist{ID: "12", Name: "乙"},
		Tracks: []music.Track{{ID: "2", Name: "起风了", Artists: []music.Artist{{ID: "13", Name: "丙"}}, Duration: 200000}},
	}}
	tt := ui.NewTester(a.view, 1200, 800)
	a.nowPlaying = true
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()

	// The second of two artists credited.
	if err := tt.Click("乙"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.nowPlaying || a.route != (route{page: pageArtist, src: "fake", id: "12"}) {
		t.Fatalf("after the click: Now Playing open %v, page %+v", a.nowPlaying, a.route)
	}

	tt.Advance(time.Second)
	tt.Frame()
	// Their name, at the start of the line it fills.
	r, ok := tt.Find("丙")
	if !ok {
		t.Fatal("no artist in the row")
	}
	tt.ClickAt(r.X+4, r.Y+r.H/2)
	tt.Frame()
	if a.route != (route{page: pageArtist, src: "fake", id: "13"}) {
		t.Fatalf("a row's artist opened %+v", a.route)
	}
	if cur := a.pl.Current(); cur == nil || cur.ID != duet.ID {
		t.Errorf("a click on a row's artist played %+v", cur)
	}

	// Back goes to the artist before.
	tt.Key(0, ui.KeyBack)
	tt.Frame()
	if a.route.id != "12" {
		t.Errorf("back went to %+v", a.route)
	}
}

// An artist's page lists their first 10 songs, playing on through all of them
// from the one clicked, and the rest once asked for; then their albums, and
// apart from these their singles and EPs, each opening its page.
func TestArtistSections(t *testing.T) {
	a := newTestApp()
	var songs []music.Track
	for i := range 14 {
		songs = append(songs, song(fmt.Sprint(i+1), fmt.Sprintf("歌%d", i+1)))
	}
	a.artists["12"] = &artistData{page: &music.ArtistPage{
		Artist: music.Artist{ID: "12", Name: "乙"},
		Tracks: songs,
		Albums: []music.Album{{ID: "8", Name: "迷跡波", Kind: "album"}, {ID: "9", Name: "春日影", Kind: "single"}},
	}}
	a.open(route{page: pageArtist, id: "12"})
	tt := ui.NewTester(a.view, 1200, 2400)
	tt.Frame()
	if _, ok := tt.Find("歌10"); !ok {
		t.Fatal("the 10th song doesn't show")
	}
	if _, ok := tt.Find("歌11"); ok {
		t.Fatal("more than 10 songs show before they're asked for")
	}
	albums, ok := tt.Find("迷跡波")
	single, ok2 := tt.Find("春日影")
	singles, ok3 := tt.Find("Singles & EPs")
	if !ok || !ok2 || !ok3 || !(albums.Y < singles.Y && singles.Y < single.Y) {
		t.Fatalf("album at %v, singles' heading at %v, single at %v", albums, singles, single)
	}

	if err := tt.Click("歌3"); err != nil {
		t.Fatal(err)
	}
	if cur := a.pl.Current(); cur == nil || cur.ID != "3" || len(a.pl.Queue()) != len(songs) {
		t.Fatalf("playing %+v of %d", cur, len(a.pl.Queue()))
	}
	if err := tt.Click("Show All 14 Songs"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("歌14"); !ok {
		t.Error("the rest don't show once asked for")
	}

	if err := tt.Click("春日影"); err != nil {
		t.Fatal(err)
	}
	if a.route != (route{page: pageAlbum, src: "fake", id: "9"}) {
		t.Fatalf("the single opened %+v", a.route)
	}
}

// Follow follows an artist at once, and asks the source to; a second click
// waits for the first, and a follow the source refuses goes back.
func TestFollowArtist(t *testing.T) {
	asked := make(chan bool, 4)
	var answer error
	a := newTestApp()
	ch := posts(a)
	fake(a).follow = func(id string, on bool) error { asked <- on; return answer }
	a.account = &music.Account{ID: "1"}
	a.artists["12"] = &artistData{page: &music.ArtistPage{Artist: music.Artist{ID: "12", Name: "乙"}}}
	a.open(route{page: pageArtist, id: "12"})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click("Follow"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	artist := &a.artists["12"].page.Artist
	if !artist.Following || !wait(t, asked, "the follow") {
		t.Fatal("not followed")
	}
	if err := tt.Click("Following"); err != nil {
		t.Fatal(err)
	}
	if len(asked) > 0 || !artist.Following {
		t.Error("unfollowed before the follow landed")
	}
	runPost(t, ch)
	tt.Frame()

	answer = errors.New("offline")
	if err := tt.Click("Following"); err != nil {
		t.Fatal(err)
	}
	if artist.Following || wait(t, asked, "the unfollow") {
		t.Fatal("not unfollowed at once")
	}
	runPost(t, ch)
	if !artist.Following {
		t.Error("an unfollow the source refused stayed")
	}
}
