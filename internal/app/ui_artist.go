package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/i18n"
	"mygo-music/internal/music"
)

// artistLinks shows a track's artists as its credits read, each a link to
// their page in color, brightening to hover under the pointer, and then,
// with album, a link to its album, as far as the source has these pages. An
// artist credited by name alone is plain text. An episode's host and show, in
// its artists' and album's place, link to the show's page.
func (a *app) artistLinks(c *ui.Context, tr *music.Track, color, hover ui.Color, album ...bool) ui.Element {
	lib := a.trackLib(tr)
	show := route{page: pageShow, src: lib.id, id: tr.ShowID}
	links := tr.ShowID != "" && lib.can("show")
	return ui.RichText(c).TextColor(color).Children(func() {
		for i, ar := range tr.Artists {
			if i > 0 {
				ui.Text(c, ", ")
			}
			switch {
			case links:
				a.link(c.Key(i), ar.Name, color, hover, show)
			case ar.ID != "" && lib.can("artist"):
				a.link(c.Key(i), ar.Name, color, hover, route{page: pageArtist, src: lib.id, id: ar.ID})
			default:
				ui.Text(c, ar.Name)
			}
		}
		if len(album) > 0 && album[0] && tr.Album.Name != "" {
			if len(tr.Artists) > 0 {
				ui.Text(c, " — ")
			}
			switch {
			case links:
				a.link(c.Key("album"), tr.Album.Name, color, hover, show)
			case tr.Album.ID != "" && lib.can("album"):
				a.link(c.Key("album"), tr.Album.Name, color, hover, route{page: pageAlbum, src: lib.id, id: tr.Album.ID})
			default:
				ui.Text(c, tr.Album.Name)
			}
		}
	})
}

// link is text in a RichText that opens a page.
func (a *app) link(c *ui.Context, text string, color, hover ui.Color, r route) {
	link := ui.Link(c, text, "").TextColor(color)
	if link.Hovered() {
		link.TextColor(hover)
	}
	// Once built: a song's row is in a list, which a new page resets.
	link.OnClick(func() { a.openPage(r) })
}

// openPage opens a page in the main window, from wherever a link to it
// shows: over Now Playing, which closes, or the notch.
func (a *app) openPage(r route) {
	a.open(r)
	a.showMain()
	a.redraw()
}

// openArtist opens an artist's page in the main window.
func (a *app) openArtist(src, id string) { a.openPage(route{page: pageArtist, src: src, id: id}) }

// artistData is what the app has of an artist: who they are, with their songs
// and albums.
type artistData struct {
	page      *music.ArtistPage
	loading   bool
	err       string
	following bool // a follow or unfollow is on its way
	allSongs  bool // their page lists all their songs, not the first few
}

// artistTopSongs is how many of an artist's songs their page lists until the
// listener asks for all.
const artistTopSongs = 10

// singleOrEP reports whether a source says an album is a single or an EP,
// which an artist's page lists apart from their albums.
func singleOrEP(al *music.Album) bool { return al.Kind == "single" || al.Kind == "ep" }

func (a *app) loadArtist(lib *library, id string) {
	d := lib.artists[id]
	if d == nil {
		d = &artistData{}
		lib.artists[id] = d
	}
	if d.page != nil || d.loading {
		return
	}
	d.loading, d.err = true, ""
	go func() {
		ar, err := answered(lib.src.Artist(id))
		a.post(func() {
			d.loading = false
			if err != nil {
				d.err = a.problem(lib, err, T("Couldn't load this artist"))
				return
			}
			d.page = ar
		})
	}()
}

// toggleFollow follows an artist, or unfollows them, at once, going back if
// the source refuses. Signed out, it asks the listener to sign in.
func (a *app) toggleFollow(lib *library, d *artistData) {
	if d.page == nil || d.following {
		return
	}
	if lib.signedOut() {
		a.promptSignIn(lib)
		return
	}
	ar := &d.page.Artist
	follow := !ar.Following
	ar.Following, d.following = follow, true
	id := ar.ID
	go func() {
		err := lib.src.FollowArtist(id, follow)
		a.post(func() {
			d.following = false
			switch {
			case err != nil:
				ar.Following = !follow
				a.actionFailed(lib, err, T("That didn't work. Try again later."))
			case follow:
				a.toast = T("Following")
			default:
				a.toast = T("Unfollowed")
			}
		})
	}()
}

// artistPage shows who an artist is over their top songs, then their albums,
// and then their singles and EPs.
func (a *app) artistPage(c *ui.Context) {
	id := a.route.id
	lib := a.library
	d := a.artists[id]
	if d == nil {
		a.loadArtist(lib, id)
		d = a.artists[id]
	}
	if d.page == nil {
		if !d.loading {
			a.centered(c, d.err, func() { a.loadArtist(lib, id) })
			return
		}
		s := a.pageShimmer(c)
		a.trackSkeleton(c, s, true, func(c *ui.Context) { a.artistHeaderSkeleton(c, s) })
		return
	}
	songs := d.page.Tracks
	var albums, singles []music.Album
	for _, al := range d.page.Albums {
		if singleOrEP(&al) {
			singles = append(singles, al)
		} else {
			albums = append(albums, al)
		}
	}
	a.scrollPage(c, func() {
		a.artistHeader(c, d)
		if len(songs) > 0 {
			a.section(c, T("Top Songs"))
			shown := len(songs)
			if !d.allSongs {
				shown = min(shown, artistTopSongs)
			}
			// The whole list plays on from the one clicked, shown or not.
			for i := range shown {
				a.trackRow(c, songs, i, true, "")
			}
			if shown < len(songs) {
				ui.Row(c).Margin(10, 0, 0, 0).Children(func() {
					if pillButton(c.Key("all-songs"), nil, Tf("Show All %d Songs", len(songs)), a.pal.hover.Over(a.pal.bg), a.pal.text).Clicked() {
						d.allSongs = true
					}
				})
			}
		}
		if len(albums) > 0 {
			a.section(c, T("Albums"))
			a.albumGrid(c.Key("albums"), albums)
		}
		if len(singles) > 0 {
			a.section(c, T("Singles & EPs"))
			a.albumGrid(c.Key("singles"), singles)
		}
		if len(songs) == 0 && len(d.page.Albums) == 0 {
			a.empty(c, T("No songs"))
		}
	})
}

// scrollPage scrolls a page of cards and rows, inset as one of tracks is:
// a tab of a page that lists tracks puts its header in the same place.
func (a *app) scrollPage(c *ui.Context, content func()) {
	top, bottom := a.insets(c)
	ui.Scroll(c.Key(a.route)).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad-12, bottom+32, pagePad-12).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		content()
	})
}

// empty says a list has nothing in it.
func (a *app) empty(c *ui.Context, msg string) {
	ui.Text(c, msg).FontSize(13).TextColor(a.pal.muted).Padding(40, 0).AlignSelf(ui.Center)
}

// artistPhotoSize is the width of an artist's round photo on their page.
const artistPhotoSize = 212

func (a *app) artistHeader(c *ui.Context, d *artistData) {
	p := a.pal
	ar := &d.page.Artist
	songs := d.page.Tracks
	lib := a.library
	ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 8, 0).Children(func() {
		ui.Box(c).Size(artistPhotoSize, artistPhotoSize).Radius(artistPhotoSize/2).Shadow(0, 18, 40, -10, p.cardShade).Children(func() {
			a.coverFill(c, ar.ImageURL, 600, artistPhotoSize/2)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			ui.Text(c, T("Artist")).FontSize(12).FontWeight(600).TextColor(p.accent)
			ui.Text(c, ar.Name).FontSize(30).FontWeight(800).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			var meta []string
			if len(songs) > 0 {
				meta = append(meta, i18n.Songs(len(songs)))
			}
			if n := len(d.page.Albums); n == 1 {
				meta = append(meta, T("1 album"))
			} else if n > 1 {
				meta = append(meta, Tf("%d albums", n))
			}
			ui.Text(c, strings.Join(meta, " · ")).FontSize(13).TextColor(p.muted)
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				if len(songs) > 0 {
					a.playButtons(c, songs)
				}
				if a.can("followArtist") {
					icon, label := iconPlus, T("Follow")
					if ar.Following {
						icon, label = iconCheck, T("Following")
					}
					if pillButton(c.Key("follow"), icon, label, p.hover.Over(p.bg), p.text).Clicked() {
						a.toggleFollow(lib, d)
					}
				}
			})
		})
	})
}

func (a *app) artistHeaderSkeleton(c *ui.Context, s shimmer) {
	ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 8, 0).Children(func() {
		s.bone(c, artistPhotoSize/2).Size(artistPhotoSize, artistPhotoSize)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			s.text(c, 12, 1.3).Width(24)
			s.text(c, 30, 1.2).WidthPercent(32)
			s.text(c, 13, 1.3).Width(160)
			ui.Column(c).Margin(2, 0, 0, 0).Children(func() {
				s.text(c, 12.5, 1.45).WidthPercent(78)
				s.text(c, 12.5, 1.45).WidthPercent(52)
			})
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				s.bone(c, 18).Size(92, 36)
				s.bone(c, 18).Size(120, 36)
			})
		})
	})
	a.section(c, T("Top Songs"))
}
