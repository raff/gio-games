package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell"
)

const (
	up    = '\u2b06' // '\u2191'
	down  = '\u2b07' // '\u2193'
	left  = '\u2b05' // '\u2190'
	right = '\u2b95' // '\u2192'
	empty = ' '

	sx = 2
	sy = 2
	cw = 2
	ch = 1
)

var (
	width  = 20
	height = 20

	dirs = []rune{up, down, left, right}
	game Game

	defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	revStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGreen)
)

type Game struct {
	screen     [][]rune
	width      int
	height     int
	cellwidth  int
	cellheight int
	count      int
	removed    int
	moves      int
}

func (g *Game) Setup(w, h, cw, ch int) {
	g.screen = nil
	g.width = w
	g.height = h
	g.cellwidth = cw
	g.cellheight = ch
	g.count = 0
	g.removed = 0
	g.moves = 0

	for i := 0; i < g.height; i++ {
		var line []rune

		for j := 0; j < g.width; j++ {
			cell := dirs[rand.Intn(len(dirs))]

			if i == 0 || i == g.height-1 || j == 0 || j == g.width-1 {
				// empty cell at the border, to make it easier to check if we can move
				cell = empty
			} else {
				g.count++
			}

			line = append(line, cell)
		}

		g.screen = append(g.screen, line)
	}
}

func (g *Game) Shuffle() {
	for y, row := range game.screen {
		for x, col := range row {
			if col != empty {
				game.screen[y][x] = dirs[rand.Intn(len(dirs))]
			}
		}
	}
}

func (g *Game) Coords(x, y int) (int, int, bool) {
	x /= g.cellwidth
	y /= g.cellheight

	if x > 0 && x < g.width-1 && y > 0 && y < g.height-1 {
		return x, y, true
	}

	return -1, -1, false
}

func (g *Game) ScreenCoords(sx, sy, x, y int) (int, int) {
	return sx + (x * g.cellwidth), sy + (y * g.cellheight)
}

func (g *Game) Update(x, y int, pressed bool) (int, int, bool) {
	cx, cy, ok := g.Coords(x, y)
	if !ok {
		return -1, -1, false
	}

	if pressed {
		var px, py int

		switch g.screen[cy][cx] {
		case up:
			for py = cy; py > 0 && g.screen[py-1][cx] == empty; py-- {
			}

			if py != cy {
				g.screen[cy][cx] = empty
				g.moves++

				if py == 0 {
					g.count--
					g.removed++
				} else {
					g.screen[py][cx] = up
				}
			}

		case down:
			for py = cy; py < g.height-1 && g.screen[py+1][cx] == empty; py++ {
			}

			if py != cy {
				g.screen[cy][cx] = empty
				g.moves++

				if py == g.height-1 {
					g.count--
					g.removed++
				} else {
					g.screen[py][cx] = down
				}
			}

		case left:
			for px = cx; px > 0 && g.screen[cy][px-1] == empty; px-- {
			}

			if px != cx {
				g.screen[cy][cx] = empty
				g.moves++

				if px == 0 {
					g.count--
					g.removed++
				} else {
					g.screen[cy][px] = left
				}
			}

		case right:
			for px = cx; px < g.width-1 && g.screen[cy][px+1] == empty; px++ {
			}

			if px != cx {
				g.screen[cy][cx] = empty
				g.moves++

				if px == g.width-1 {
					g.count--
					g.removed++
				} else {
					g.screen[cy][px] = right
				}
			}
		}
	}

	return cx, cy, true
}

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
	x2 := x1 + (game.width * 2) + 1
	y2 := y1 + game.height + 1
	style := boxStyle

	// Fill screen
	for y, row := range game.screen {
		for x, col := range row {
			s.SetContent(x1+(2*x)+1, y1+y+1, col, nil, style)
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
	msg := "                                       "

	if cx, cy, ok = game.Update(x-sx-1, y-sy-1, pressed); ok {
		s.ShowCursor(game.ScreenCoords(sx+1, sy+1, cx, cy))

		msg = fmt.Sprintf("moves=%v remain=%v removed=%v x=%v y=%v  ",
			game.moves, game.count, game.removed, cx, cy)
	}

	drawScreen(s)
	drawText(s, sx, sy+height+2, sx+len(msg)+1, sy+height+2, boxStyle, msg)
	return
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

	rand.Seed(time.Now().Unix())

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
	game.Setup(width, height, cw, ch)
	drawScreen(s)

	// Event loop
	quit := func() {
		s.Fini()
		os.Exit(0)
	}

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
			} else if crune == ' ' {
				checkScreen(s, cx, cy, true)
			} else if crune == 'R' || crune == 'r' {
				game.Setup(width, height, cw, ch)
				s.Clear()
				drawScreen(s)
			} else if crune == 'S' || crune == 's' {
				game.Shuffle()
				s.Clear()
				drawScreen(s)
			}
		case *tcell.EventMouse:
			cx, cy = ev.Position()
			button := ev.Buttons() & tcell.ButtonMask(0xff)
			checkScreen(s, cx, cy, button != tcell.ButtonNone)
		}
	}
}
