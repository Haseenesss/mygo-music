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
	At     time.Time        `json:"at"`
	Listed []plugin.Listing `json:"listed"`
}

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
	if json.Unmarshal(data, &f) == nil && f.Listed != nil {
		s.listed, s.at = f.Listed, f.At
	}
}

// write keeps what GitHub listed, for the next launch.
func (s *pluginStore) write() {
	if s.file == "" {
		return
	}
	data, err := json.Marshal(storeFile{At: s.at, Listed: s.listed})
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

// storeSettings is the plugins' Discover tab: the plugins on GitHub with the topic, each
// found by a search of the settings by its name or what it says it does.
func (a *app) storeSettings(c *ui.Context, block settingsBlock, p palette) {
	s := &a.store
	block(T("Plugins on GitHub")+" "+plugin.Topic, func() {
		if !s.asked && !s.busy {
			a.loadStore(false)
		}
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
	})
	var official, community []plugin.Listing
	for _, l := range s.listed {
		if plugin.Official(l.Repo) {
			official = append(official, l)
		} else {
			community = append(community, l)
		}
	}
	if len(official) > 0 {
		block(T("Official Plugins"), func() {
			a.storeHeading(c.Key("official"), T("Official Plugins"), T("Made by the author of MyGo Music"), p)
		})
		a.storeRows(c, block, official, p)
	}
	if len(community) > 0 {
		block(T("Community Plugins"), func() {
			a.storeHeading(c.Key("community"), T("Community Plugins"), T("Made by others, and not reviewed. See what each asks for before you install it."), p)
		})
		a.storeRows(c, block, community, p)
	}
}

// storeHeading heads a part of the store's list.
func (a *app) storeHeading(c *ui.Context, title, desc string, p palette) {
	ui.Column(c).Padding(14, 16, 10, 16).Gap(3).Background(p.hover.Alpha(0.5)).Children(func() {
		ui.Text(c, title).FontSize(12.5).FontWeight(600)
		ui.Text(c, desc).FontSize(11.5).TextColor(p.muted).LineHeight(1.4)
	})
}

// storeRows are the store's plugins, each with a way to install it.
func (a *app) storeRows(c *ui.Context, block settingsBlock, listed []plugin.Listing, p palette) {
	in := &a.installer
	for _, l := range listed {
		block(l.Repo+" "+l.Description, func() {
			c := c.Key("listed-" + l.Repo)
			_, name, _ := strings.Cut(l.Repo, "/")
			ui.Row(c).Padding(12, 16).Gap(12).AlignItems(ui.Center).MinHeight(48).Children(func() {
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
					ui.Text(c, name).FontSize(13.5).FontWeight(500).SingleLine()
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
		})
	}
}
