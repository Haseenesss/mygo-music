package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/plugin"
)

// The tabs of the plugins' settings.
const (
	pluginsInstalled = iota
	pluginsDiscover
)

// pluginTabs switches the plugins' settings between those installed and those
// on GitHub to discover.
func (a *app) pluginTabs(c *ui.Context) {
	ui.Row(c).Justify(ui.Center).Margin(0, 0, 6, 0).Children(func() {
		segmented(c.Key("plugin-tabs"), &a.pluginTab, T("Installed"), T("Discover"))
	})
}

// listPlugins lists the plugins on GitHub, by their topic; tests replace it.
var listPlugins = plugin.Listed

// storeFresh is how long what GitHub listed serves before the store asks it
// again by itself, kept between launches: its search allows 10 requests a
// minute to a listener without a token, shared by all on the same network.
// Refresh asks at once.
const storeFresh = 6 * time.Hour

// pluginStore is what the store has of the plugins on GitHub: what it listed
// last, from the store's file, and anew once that's old, or the listener asks.
type pluginStore struct {
	listed []plugin.Listing
	at     time.Time // when GitHub listed them
	asked  bool      // they were asked for, or came from the file fresh
	busy   bool
	err    string
	file   string // where they're kept between launches; none in tests
}

// storeFile is what the store keeps of what GitHub listed.
type storeFile struct {
	// Version is storeVersion as the list was kept; one kept otherwise, as
	// before official plugins were found by name, is asked for anew.
	Version int              `json:"version,omitempty"`
	At      time.Time        `json:"at"`
	Listed  []plugin.Listing `json:"listed"`
}

// storeVersion is the version of what the store keeps.
const storeVersion = 2

// loadStore shows what GitHub listed last, and asks it again if that's old,
// or if again is set, as Refresh does. What went wrong asking leaves what was
// listed before.
func (a *app) loadStore(again bool) {
	s := &a.store
	if s.busy {
		return
	}
	if s.listed == nil {
		s.read()
	}
	if !again && s.listed != nil && time.Since(s.at) < storeFresh {
		s.asked = true
		return
	}
	s.busy, s.err = true, ""
	go func() {
		listed, err := listPlugins()
		a.post(func() {
			s.busy, s.asked = false, true
			if err != nil {
				s.err = installError(err)
				return
			}
			s.listed, s.at = listed, time.Now()
			s.write()
		})
	}()
}

// read takes what the store kept, if it kept anything.
func (s *pluginStore) read() {
	if s.file == "" {
		return
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	var f storeFile
	if json.Unmarshal(data, &f) == nil && f.Version == storeVersion && f.Listed != nil {
		s.listed, s.at = f.Listed, f.At
	}
}

// write keeps what GitHub listed, for the next launch.
func (s *pluginStore) write() {
	if s.file == "" {
		return
	}
	data, err := json.Marshal(storeFile{Version: storeVersion, At: s.at, Listed: s.listed})
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(s.file), 0o755)
	os.WriteFile(s.file, data, 0o644)
}

// installedFrom is the library of the plugin installed from repo, if one is.
func (a *app) installedFrom(repo string) *library {
	for _, lib := range a.allLibs {
		if plug := pluginOf(lib); plug != nil {
			if o := plug.Origin(); o != nil && strings.EqualFold(o.Repo, repo) {
				return lib
			}
		}
	}
	return nil
}

// installListed downloads a plugin of the store's, to ask about it as one
// written in From GitHub… is.
func (a *app) installListed(repo string) {
	in := &a.installer
	in.repo = repo
	in.open, in.github = true, true
	a.installFromRepo()
}

// storeOfficial reports whether a plugin listed is MyGo Music's own; a list
// kept by an older version says so by its owner alone.
func storeOfficial(l plugin.Listing) bool { return l.Official || plugin.Official(l.Repo) }

// storePage is the plugins' Discover tab: what the store asked GitHub, then
// MyGo Music's own plugins and the community's, each on a card of its own
// under its heading.
func (a *app) storePage(c *ui.Context, p palette) {
	s := &a.store
	if !s.asked && !s.busy {
		a.loadStore(false)
	}
	var official, community []plugin.Listing
	for _, l := range s.listed {
		if storeOfficial(l) {
			official = append(official, l)
		} else {
			community = append(community, l)
		}
	}
	card := func(key string, rows func()) {
		ui.Column(c.Key(key)).Radius(12).Background(p.bg).Border(1, p.hairline).Dividers(1, p.hairline).Children(rows)
	}
	card("store-status", func() { a.storeStatus(c, p) })
	if len(official) > 0 {
		a.storeHeading(c, T("Official Plugins"), T("Made by the author of MyGo Music"), len(official), p)
		card("store-official", func() {
			for _, l := range official {
				a.storeRow(c, l, p)
			}
		})
	}
	if len(community) > 0 {
		a.storeHeading(c, T("Community Plugins"), T("Made by others, and not reviewed. See what each asks for before you install it."), len(community), p)
		card("store-community", func() {
			for _, l := range community {
				a.storeRow(c, l, p)
			}
		})
	}
}

// storeSettings are the store's plugins as a search of the settings finds
// them, by their names or what they say they do.
func (a *app) storeSettings(c *ui.Context, block settingsBlock, p palette) {
	block(T("Plugins on GitHub")+" "+plugin.Topic, func() { a.storeStatus(c, p) })
	for _, l := range a.store.listed {
		block(l.Repo+" "+l.Description, func() { a.storeRow(c, l, p) })
	}
}

// storeStatus says where the store's plugins come from, or that they're on
// their way, or what went wrong asking, with a way to ask again.
func (a *app) storeStatus(c *ui.Context, p palette) {
	s := &a.store
	ui.Row(c).Padding(12, 16).Gap(12).AlignItems(ui.Center).MinHeight(48).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
			ui.Text(c, T("Plugins on GitHub")).FontSize(13.5).FontWeight(500)
			desc := Tf("Repositories with the topic %s", plugin.Topic)
			switch {
			case s.busy:
				desc = T("Looking on GitHub…")
			case s.err != "":
				desc = s.err
			}
			ui.Text(c, desc).FontSize(12).TextColor(p.muted).LineHeight(1.4)
		})
		if ui.Button(c.Key("refresh-store"), T("Refresh")).Disabled(s.busy).Clicked() {
			a.loadStore(true)
		}
	})
}

// storeHeading heads a card of the store's plugins, outside it, with how many
// it has.
func (a *app) storeHeading(c *ui.Context, title, desc string, n int, p palette) {
	ui.Column(c).Margin(20, 4, 6, 4).Gap(3).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, title).FontSize(14).FontWeight(700)
			ui.Text(c, count(n)).FontSize(12).TextColor(p.muted).FontFeatures("tnum")
		})
		ui.Text(c, desc).FontSize(12).TextColor(p.muted).LineHeight(1.4)
	})
}

// storeRow is a plugin of the store's, with a way to install it, or what's
// become of it.
func (a *app) storeRow(c *ui.Context, l plugin.Listing, p palette) {
	in := &a.installer
	c = c.Key("listed-" + l.Repo)
	_, name, _ := strings.Cut(l.Repo, "/")
	ui.Row(c).Padding(12, 16).Gap(12).AlignItems(ui.Center).MinHeight(48).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
				ui.Text(c, name).FontSize(13.5).FontWeight(500).SingleLine().Shrink(1)
				if storeOfficial(l) {
					ui.Box(c).Padding(1, 6).Radius(4).Background(p.accent.Alpha(0.12)).Shrink(0).Children(func() {
						ui.Text(c, T("Official")).FontSize(10.5).FontWeight(600).TextColor(p.accent)
					})
				}
			})
			ui.Row(c).Gap(8).MinWidth(0).Children(func() {
				ui.Link(c, l.Repo, "https://github.com/"+l.Repo).FontSize(12).TextColor(p.muted).SingleLine().Shrink(1).Tooltip(T("Open on GitHub"))
				if l.Stars > 0 {
					ui.Text(c, "★ "+count(l.Stars)).FontSize(12).TextColor(p.muted).SingleLine().Shrink(0)
				}
			})
			if l.Description != "" {
				ui.Text(c, l.Description).FontSize(12).TextColor(p.muted).LineHeight(1.4)
			}
		})
		switch {
		case a.installedFrom(l.Repo) != nil:
			ui.Text(c, T("Installed")).FontSize(12.5).TextColor(p.muted).Shrink(0)
		case in.busy && strings.EqualFold(in.repo, l.Repo):
			ui.Text(c, T("Downloading…")).FontSize(12.5).TextColor(p.muted).Shrink(0)
		default:
			if ui.Button(c.Key("install"), T("Install")).Label(Tf("Install %s", l.Repo)).Disabled(in.busy).Clicked() {
				a.installListed(l.Repo)
			}
		}
	})
}
