package app

import (
	"strings"

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

// pluginStore is what the store has of the plugins on GitHub, asked for once the
// listener first looks, and again when they ask.
type pluginStore struct {
	listed []plugin.Listing
	asked  bool // they were asked for, and came or failed
	busy   bool
	err    string
}

// loadStore asks GitHub for the plugins with the topic.
func (a *app) loadStore() {
	s := &a.store
	if s.busy {
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
			s.listed = listed
		})
	}()
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
			a.loadStore()
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
				a.loadStore()
			}
		})
	})
	in := &a.installer
	for _, l := range s.listed {
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
