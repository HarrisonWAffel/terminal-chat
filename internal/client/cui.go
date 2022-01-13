package client

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/pion/webrtc/v3"
	"log"
)

type GUI struct {
	InputChan   chan string
	OutputChan  chan string
	NetworkChan chan string
	Username    string
	PeerConn    *webrtc.PeerConnection
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

var textBoxSelected = true

func nextView(g *gocui.Gui, v *gocui.View) error {
	switch textBoxSelected {
	case true:
		g.SetCurrentView("bottom")
	case false:
		g.SetCurrentView("main")
	}
	textBoxSelected = !textBoxSelected
	return nil
}

func (gui *GUI) getInput(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	l, err := v.Line(cy)
	if err != nil {
		l = ""
	}

	gui.InputChan <- gui.Username + ": " + l
	gui.NetworkChan <- gui.Username + ": " + l

	err = v.SetCursor(0, 0)
	if err != nil {
		return err
	}

	v.Clear()
	return nil
}

func (gui *GUI) keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("main", gocui.KeyCtrlSpace, gocui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("bottom", gocui.KeyCtrlSpace, gocui.ModNone, nextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("bottom", gocui.KeyEnter, gocui.ModNone, gui.getInput); err != nil {
		return err
	}
	return nil
}

func (gui *GUI) layout(g *gocui.Gui) error {
	sideBarSize := 20
	maxX, maxY := g.Size()
	if v, err := g.SetView("side", 0, 0, sideBarSize, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Terminal Chat"

		stats, _ := gui.PeerConn.GetStats().GetConnectionStats(gui.PeerConn)
		fmt.Fprintln(v, stats.Type+"\n")
		fmt.Fprintln(v, "Username:\n"+gui.Username+"\n")
		fmt.Fprintf(v, "# Channels: %d\n", stats.DataChannelsOpened)
	}

	if v, err := g.SetView("main", sideBarSize, 0, maxX-1, maxY-6); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.Wrap = true
		v.Title = "Chat Room"
		v.Highlight = true
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		fmt.Fprintf(v, "Connection Successful!\n")
		fmt.Fprintf(v, "Connected to Peer...\n\n")
	}

	if v, err := g.SetView("bottom", sideBarSize, maxY-6, maxX-1, maxY-1); err != nil {
		v.Editable = true
		v.Title = "Type Here"
		v.Wrap = true
		if _, err := g.SetCurrentView("bottom"); err != nil {
			return err
		}
	}

	return nil
}

func (gui *GUI) StartGUI() error {

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(gui.layout)

	if err := gui.keybindings(g); err != nil {
		log.Panicln(err)
	}

	go func() {
		for {
			select {
			case msg := <-gui.InputChan:
				g.Update(func(gui *gocui.Gui) error {
					v, err := gui.View("main")
					if err != nil {
						return err
					}
					fmt.Fprintf(v, "%s\n", msg)
					return nil
				})
			case msg := <-gui.OutputChan:
				g.Update(func(cgui *gocui.Gui) error {
					v, err := cgui.View("main")
					if err != nil {
						return err
					}
					fmt.Fprintf(v, "%s\n", msg)
					return nil
				})
			}
		}
	}()

	return g.MainLoop()
}
