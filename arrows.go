package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell"
	"github.com/raff/arrows/game"
)

const (
	up    = '\u2b06' // '\u2191'
	down  = '\u2b07' // '\u2193'
	left  = '\u2b05' // '\u2190'
	right = '\u2b95' // '\u2192'
	empty = ' '

	cw = 2
	ch = 1
)

var (
	width  = 20
	height = 20

	sx = 2
	sy = 2

	dirs  = []rune{empty, up, down, left, right}
	agame game.Game

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	revStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGreen)
)

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawScreen(s tcell.Screen) {
	x1 := sx
	y1 := sy
	x2 := x1 + (agame.Width * 2) + 1
	y2 := y1 + agame.Height + 1
	style := boxStyle

	// Fill screen
	for y, row := range agame.Screen {
		for x, col := range row {
			s.SetContent(x1+(2*x)+1, y1+y+1, dirs[col], nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}
}

func checkScreen(s tcell.Screen, x, y int, pressed bool) (cx, cy int, ok bool) {
	msg := "                                "

	if cx, cy, ok = agame.Update(x-sx-1, y-sy-1, pressed, true); ok {
		s.ShowCursor(agame.ScreenCoords(sx+1, sy+1, cx, cy))

		msg = fmt.Sprintf("moves=%v remain=%v removed=%v   ", agame.Moves, agame.Count, agame.Removed)
	}

	drawScreen(s)
	drawText(s, sx, sy+height+2, sx+len(msg)+1, sy+height+2, boxStyle, msg)
	return
}

func centerScreen(s tcell.Screen) (int, int, bool) {
	gw, gh := agame.Width*2+2, agame.Width+2
	w, h := s.Size()

	px, py := sx, sy

	if w > gw {
		sx = (w - gw) / 2 // center horizontally
	}

	if h > gh {
		sy = (h - gh) / 2 // center vertically
	}

	if sx != px || sy != py {
		s.Clear()
		return agame.Coords(px, py)
	}

	return -1, -1, false
}

func main() {
	flag.IntVar(&width, "width", width, "screen width")
	flag.IntVar(&height, "height", height, "screen height")
	flag.Parse()

	if width <= 0 || height <= 0 {
		log.Fatal("invalid width or height")
	}

	width += 2  // add border
	height += 2 // to simplify boundary checks

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.Clear()

	// Draw initial screen
	agame.Setup(width, height, cw, ch)
	drawScreen(s)

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	cx, cy := agame.ScreenCoords(sx+1, sy+1, 1, 1)
	s.ShowCursor(cx, cy)

	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()

			if x, y, ok := centerScreen(s); ok {
				cx, cy = agame.ScreenCoords(sx+1, sy+1, x, y)
				s.ShowCursor(cx, cy)
				drawScreen(s)
			}

		case *tcell.EventKey:
			ckey, crune := ev.Key(), ev.Rune()

			if ckey == tcell.KeyEscape || ckey == tcell.KeyCtrlC {
				quit()
			} else if ckey == tcell.KeyCtrlL {
				s.Sync()
			} else if ckey == tcell.KeyUp {
				if _, _, ok := checkScreen(s, cx, cy-1, false); ok {
					cy--
				}
			} else if ckey == tcell.KeyDown {
				if _, _, ok := checkScreen(s, cx, cy+1, false); ok {
					cy++
				}
			} else if ckey == tcell.KeyLeft {
				if _, _, ok := checkScreen(s, cx-2, cy, false); ok {
					cx -= 2
				}
			} else if ckey == tcell.KeyRight {
				if _, _, ok := checkScreen(s, cx+2, cy, false); ok {
					cx += 2
				}
			} else if crune == ' ' { // hit
				checkScreen(s, cx, cy, true)
			} else if crune == 'U' || crune == 'u' { // undo
				if x, y, ok := agame.Undo(); ok {
					cx, cy = agame.ScreenCoords(sx+1, sy+1, x, y)
					s.ShowCursor(cx, cy)
					drawScreen(s)
				}
			} else if crune == 'R' || crune == 'r' { // reset
				agame.Setup(width, height, cw, ch)
				drawScreen(s)
			} else if crune == 'S' || crune == 's' { // reshuffle
				agame.Shuffle()
				drawScreen(s)
			} else if crune == 'H' || crune == 'h' { // remove all "free" arrows
				for y := 1; y < agame.Height-1; y++ {
					for x := 1; x < agame.Width-1; x++ {
						x, y := agame.ScreenCoords(0, 0, x, y)
						agame.Update(x, y, true, false)
					}
				}

				checkScreen(s, cx, cy, false)
			} else if crune == 'P' || crune == 'p' { // auto play
				s.PostEvent(tcell.NewEventInterrupt(nil))
			}
		case *tcell.EventMouse:
			cx, cy = ev.Position()
			button := ev.Buttons() & tcell.ButtonMask(0xff)
			checkScreen(s, cx, cy, button != tcell.ButtonNone)

		case *tcell.EventInterrupt:
			for y := 1; y < agame.Height-1; y++ {
				for x := 1; x < agame.Width-1; x++ {
					x, y := agame.ScreenCoords(0, 0, x, y)
					agame.Update(x, y, true, false)
				}
			}

			checkScreen(s, cx, cy, false)

			if agame.Count > 0 {
				agame.Shuffle()
				time.AfterFunc(300*time.Millisecond, func() { s.PostEvent(ev) })
			}
		}
	}
}
