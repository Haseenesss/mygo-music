package app

import (
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mygo-music/internal/player"
)

// speedLabel is a speed as it's written, such as 1.5×.
func speedLabel(s float64) string { return strconv.FormatFloat(s, 'f', -1, 64) + "×" }

// setSpeed plays at speed s from now on, and keeps it for the next time the
// app opens.
func (a *app) setSpeed(s float64) {
	a.pl.SetSpeed(s)
	a.settings.Speed = a.pl.Speed()
	a.settings.save()
	if a.win != nil {
		// The Controls menu checks the speed it plays at.
		a.setMenu()
	}
	a.redraw()
}

// speedButton says how fast what's playing plays, in color, or in fg at 1,
// and its menu changes it.
func (a *app) speedButton(c *ui.Context, fg, color, hover ui.Color) {
	speed := a.pl.Speed()
	if speed == 1 {
		color = fg
	}
	b := ui.ButtonBase(c.Key("speed")).Height(28).PaddingX(6).Radius(6).Label(T("Playback Speed")).Tooltip(T("Playback Speed"))
	if b.Hovered() {
		b.Background(hover)
	}
	b.Children(func() {
		ui.Text(c, speedLabel(speed)).FontSize(12).FontWeight(600).FontFeatures("tnum").TextColor(color)
	})
	b.Menu(func(m *ui.Menu) {
		for _, s := range player.Speeds {
			if m.Item(speedLabel(s)).Checked(s == speed).Chosen() {
				a.setSpeed(s)
			}
		}
	})
}

// speedMenu is the Controls menu's speeds, the one it plays at checked.
func (a *app) speedMenu(run func(func()) func(*mygo.MenuItem, *mygo.Window)) []*mygo.MenuItem {
	items := make([]*mygo.MenuItem, len(player.Speeds))
	for i, s := range player.Speeds {
		items[i] = &mygo.MenuItem{Label: speedLabel(s), Type: mygo.MenuItemRadio, Checked: s == a.pl.Speed(), Click: run(func() { a.setSpeed(s) })}
	}
	return items
}
