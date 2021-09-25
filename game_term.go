//go:build !ios && !android && !js

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/gdamore/tcell/v2"
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
	sx = 2
	sy = 2

	dirs = []rune{empty, up, down, left, right}

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
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
	x2 := x1 + (game.Width * 2) + 1
	y2 := y1 + game.Height + 1
	style := boxStyle

	// Fill screen
	for y, row := range game.Screen {
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

func checkScreen(s tcell.Screen, x, y int, op Updates) (cx, cy int, mov Updates) {
	msg := "                                         "

	cx, cy, mov = game.Update(x-sx-1, y-sy-1, op)
	if mov != Invalid {
		s.ShowCursor(game.ScreenCoords(sx+1, sy+1, cx, cy))
		msg = fmt.Sprintf("moves=%v remain=%v removed=%v seq=%v       ", game.Moves, game.Count, game.Removed, game.Seq)
	}

	drawScreen(s)
	drawText(s, sx, sy+gameHeight+2, sx+len(msg)+1, sy+gameHeight+2, boxStyle, msg)

	return
}

func centerScreen(s tcell.Screen) (int, int, bool) {
	gw, gh := game.Width*2+2, game.Height+2
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
		return game.Coords(px, py)
	}

	return -1, -1, false
}

func termGame() {
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
	game.Setup(gameWidth, gameHeight, cw, ch)
	drawScreen(s)

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

	ops := map[bool]Updates{true: Move, false: None}

	cx, cy := game.ScreenCoords(sx+1, sy+1, 1, 1)
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
				cx, cy = game.ScreenCoords(sx+1, sy+1, x, y)
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
				if _, _, mov := checkScreen(s, cx, cy-1, None); mov != Invalid {
					cy--
				}
			} else if ckey == tcell.KeyDown {
				if _, _, mov := checkScreen(s, cx, cy+1, None); mov != Invalid {
					cy++
				}
			} else if ckey == tcell.KeyLeft {
				if _, _, mov := checkScreen(s, cx-2, cy, None); mov != Invalid {
					cx -= 2
				}
			} else if ckey == tcell.KeyRight {
				if _, _, mov := checkScreen(s, cx+2, cy, None); mov != Invalid {
					cx += 2
				}
			} else if crune == ' ' { // hit
				_, _, mov := checkScreen(s, cx, cy, Move)
				audioPlay(mov)

				if game.Count == 0 {
					if game.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				}
			} else if crune == 'U' || crune == 'u' { // undo
				if x, y, ok := game.Undo(); ok {
					audioPlay(Undo)
					cx, cy = game.ScreenCoords(sx+1, sy+1, x, y)
					checkScreen(s, cx, cx, None)
				}
			} else if crune == 'R' || crune == 'r' { // reset
				audioPlay(Undo)
				game.Setup(gameWidth, gameHeight, cw, ch)
				drawScreen(s)
			} else if crune == 'S' || crune == 's' { // reshuffle
				audioPlay(Shuffle)
				game.Shuffle(shuffleDir)
				drawScreen(s)
			} else if crune == 'H' || crune == 'h' { // remove all "free" arrows
				moved := Invalid

				for y := 1; y < game.Height-1; y++ {
					for x := 1; x < game.Width-1; x++ {
						x, y := game.ScreenCoords(0, 0, x, y)
						_, _, mov := game.Update(x, y, Remove)
						if mov > moved {
							moved = mov
						}
					}
				}

				audioPlay(moved)

				if game.Count == 0 {
					if game.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				} else if moved != None {
					game.Seq = 0
				}

				checkScreen(s, cx, cy, None)
			} else if crune == 'P' || crune == 'p' { // auto play
				s.PostEvent(tcell.NewEventInterrupt(false))
			}
		case *tcell.EventMouse:
			cx, cy = ev.Position()
			pressed := ev.Buttons()&tcell.ButtonMask(0xff) != tcell.ButtonNone
			_, _, mov := checkScreen(s, cx, cy, ops[pressed])
			if pressed {
				audioPlay(mov)

				if game.Count == 0 {
					if game.Winner() {
						s.PostEvent(tcell.NewEventInterrupt(true))
					}
				}
			}

		case *tcell.EventInterrupt:
			winner := ev.Data().(bool)
			count := game.Count

			for y := 1; y < game.Height-1; y++ {
				for x := 1; x < game.Width-1; x++ {
					x, y := game.ScreenCoords(0, 0, x, y)
					game.Update(x, y, Remove)
				}
			}

			if count == game.Count { // no changes
				break
			}

			checkScreen(s, cx, cy, None)

			if game.Count > 0 {
				if !winner {
					audioPlay(Shuffle)
					game.Shuffle(shuffleDir)
				}

				time.AfterFunc(300*time.Millisecond, func() { s.PostEvent(ev) })
			}
		}
	}
}
