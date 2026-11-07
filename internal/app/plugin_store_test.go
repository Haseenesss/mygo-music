package app

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/plugin"
)

// The plugins' Discover tab asks GitHub for its plugins once, as it's first
// shown, and offers to install those that aren't; Install downloads one to ask
// about it, as From GitHub… does.
func TestStoreInstallsWhatItLists(t *testing.T) {
	asked := make(chan bool, 4)
	wasList, wasFetch := listPlugins, fetchPlugin
	defer func() { listPlugins, fetchPlugin = wasList, wasFetch }()
	listPlugins = func() ([]plugin.Listing, error) {
		asked <- true
		return []plugin.Listing{{Repo: "O/GH", Stars: 3}, {Repo: "owner/new", Description: "New"}}, nil
	}
	fetched := make(chan string, 1)
	fetchPlugin = func(repo, dir string) (*plugin.Pending, error) {
		fetched <- repo
		return &plugin.Pending{Release: plugin.Release{Repo: repo}, Manifest: plugin.Manifest{ID: "new", Name: "New"}}, nil
	}
	a := newTestApp()
	ch := posts(a)
	addFake(a, githubPlugin(t)) // from o/gh
	a.settingsPage = "plugins"
	tt := ui.NewTester(a.settingsView, 780, 1000)
	tt.Frame()
	if err := tt.Click("Discover"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	wait(t, asked, "the list")
	runPost(t, ch)
	tt.Frame()
	tt.Frame()
	if len(asked) != 0 {
		t.Fatal("asked GitHub again as it drew")
	}

	if err := tt.Click("Install O/GH"); err == nil {
		t.Error("offers to install what's installed")
	}
	if err := tt.Click("Install owner/new"); err != nil {
		t.Fatal(err)
	}
	if got := wait(t, fetched, "the download"); got != "owner/new" {
		t.Fatalf("downloaded %q", got)
	}
	runPost(t, ch)
	if !a.installer.open || a.installer.pending == nil || a.installer.pending.Manifest.ID != "new" {
		t.Fatalf("asks about %v", a.installer.pending)
	}
}

// What went wrong asking GitHub shows, and Refresh asks again.
func TestStoreRefreshesAfterFailing(t *testing.T) {
	fail := true
	wasList := listPlugins
	defer func() { listPlugins = wasList }()
	listPlugins = func() ([]plugin.Listing, error) {
		if fail {
			return nil, errors.New("offline")
		}
		return []plugin.Listing{{Repo: "owner/new"}}, nil
	}
	a := newTestApp()
	ch := posts(a)
	a.settingsPage = "plugins"
	tt := ui.NewTester(a.settingsView, 780, 1000)
	tt.Frame()
	if err := tt.Click("Discover"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	runPost(t, ch)
	tt.Frame()
	if a.store.err != "offline" {
		t.Fatalf("says %q", a.store.err)
	}

	fail = false
	if err := tt.Click("Refresh"); err != nil {
		t.Fatal(err)
	}
	runPost(t, ch)
	tt.Frame()
	if a.store.err != "" {
		t.Errorf("still says %q", a.store.err)
	}
	if err := tt.Click("Install owner/new"); err != nil {
		t.Error(err)
	}
}
