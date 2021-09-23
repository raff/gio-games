package main

import (
	"math/rand"
	"time"
)

// Arrow directions
type Dir int8

type Updates int8

const (
	Empty = Dir(0)
	Up    = Dir(1)
	Down  = Dir(2)
	Left  = Dir(3)
	Right = Dir(4)

	DirCount = 4

	Invalid = Updates(0) // invalid coordinates
	None    = Updates(1) // cannot move
	Move    = Updates(2) // move arrow/moved
	Remove  = Updates(3) // remove arrow/removed

	// these is actually only used for playing sound effects
	Shuffle = Updates(-1)
	Undo    = Updates(-2)
)

type FromTo struct {
	X1 int
	Y1 int
	D1 Dir

	X2 int
	Y2 int
	D2 Dir
}

type Game struct {
	Screen  [][]Dir
	Width   int
	Height  int
	Count   int
	Removed int
	Moves   int
	Seq     int

	cellwidth  int
	cellheight int

	stack []FromTo
}

func (g *Game) Push(x1, y1 int, d1 Dir, x2, y2 int, d2 Dir) {
	g.stack = append(g.stack, FromTo{X1: x1, Y1: y1, D1: d1, X2: x2, Y2: y2, D2: d2})
}

func (g *Game) Pop() *FromTo {
	l := len(g.stack)

	if l == 0 {
		return nil
	}

	move := g.stack[l-1]
	g.stack = g.stack[:l-1]

	return &move
}

//
// setup game
//
func (g *Game) Setup(w, h, cw, ch int) {
	g.Screen = nil
	g.Width = w
	g.Height = h
	g.Count = 0
	g.Removed = 0
	g.Moves = 0
	g.Seq = 0

	g.cellwidth = cw
	g.cellheight = ch
	g.stack = g.stack[:0]

	rand.Seed(time.Now().Unix())

	for i := 0; i < g.Height; i++ {
		var line []Dir

		for j := 0; j < g.Width; j++ {
			cell := Dir(rand.Intn(DirCount) + 1) // 0 is Empty

			if i == 0 || i == g.Height-1 || j == 0 || j == g.Width-1 {
				// empty cell at the border, to make it easier to check if we can move
				cell = Empty
			} else {
				g.Count++
			}

			line = append(line, cell)
		}

		g.Screen = append(g.Screen, line)
	}
}

//
// shuffle arrows
// (actually replace arrows where present)
//
func (g *Game) Shuffle() {
	g.Count = 0
	g.Seq = 0

	for y, row := range g.Screen {
		for x, col := range row {
			if col != Empty {
				g.Count++

				//g.Screen[y][x] = Dir(rand.Intn(DirCount) + 1) // 0 is Empty

				switch g.Screen[y][x] {
				case Up:
					g.Screen[y][x] = Left
				case Down:
					g.Screen[y][x] = Right
				case Left:
					g.Screen[y][x] = Down
				case Right:
					g.Screen[y][x] = Up
				}
			}
		}
	}

	g.stack = g.stack[:0]
}

//
// convert screen coordinates to game coordinates
//
// returns false if screen coordinates are outside game boundary
//
func (g *Game) Coords(x, y int) (int, int, bool) {
	x /= g.cellwidth
	y /= g.cellheight

	if x > 0 && x < g.Width-1 && y > 0 && y < g.Height-1 {
		return x, y, true
	}

	return -1, -1, false
}

//
// convert game coordinates to screen coordinates
//
// x,y: game coordinates
// sx,sy: screen offset
//
func (g *Game) ScreenCoords(sx, sy, x, y int) (int, int) {
	return sx + (x * g.cellwidth), sy + (y * g.cellheight)
}

//
// update game based on screen coordinates
// returns game coordinates (and false if outside of boundaries)
//
// remove: remove arrows at x,y
// move: if not out of boundary move arrow to last empty position
//
func (g *Game) Update(x, y int, op Updates) (cx, cy int, res Updates) {
	var ok bool

	cx, cy, ok = g.Coords(x, y)
	if !ok {
		return -1, -1, Invalid
	}

	res = None

	if op > None {
		var px, py int

		switch g.Screen[cy][cx] {
		case Up:
			for py = cy; py > 0 && g.Screen[py-1][cx] == Empty; py-- {
			}

			if py != cy {
				if py == 0 {
					g.Count--
					g.Removed++
					g.Seq++
					res = Remove
				} else {
					if op != Move {
						return
					}

					g.Screen[py][cx] = Up
					res = Move
				}

				g.Push(cx, py, Empty, cx, cy, g.Screen[cy][cx])
				g.Screen[cy][cx] = Empty
				g.Moves++

			}

		case Down:
			for py = cy; py < g.Height-1 && g.Screen[py+1][cx] == Empty; py++ {
			}

			if py != cy {
				if py == g.Height-1 {
					g.Count--
					g.Removed++
					g.Seq++
					res = Remove
				} else {
					if op != Move {
						return
					}
					g.Screen[py][cx] = Down
					res = Move
				}

				g.Push(cx, py, Empty, cx, cy, g.Screen[cy][cx])
				g.Screen[cy][cx] = Empty
				g.Moves++
			}

		case Left:
			for px = cx; px > 0 && g.Screen[cy][px-1] == Empty; px-- {
			}

			if px != cx {
				if px == 0 {
					g.Count--
					g.Removed++
					g.Seq++
					res = Remove
				} else {
					if op != Move {
						return
					}

					g.Screen[cy][px] = Left
					res = Move
				}

				g.Push(px, cy, Empty, cx, cy, g.Screen[cy][cx])
				g.Screen[cy][cx] = Empty
				g.Moves++
			}

		case Right:
			for px = cx; px < g.Width-1 && g.Screen[cy][px+1] == Empty; px++ {
			}

			if px != cx {
				if px == g.Width-1 {
					g.Count--
					g.Removed++
					g.Seq++
					res = Remove
				} else {
					if op != Move {
						return
					}

					g.Screen[cy][px] = Right
					res = Move
				}

				g.Push(px, cy, Empty, cx, cy, g.Screen[cy][cx])
				g.Screen[cy][cx] = Empty
				g.Moves++
			}
		}
	}

	return
}

func (g *Game) Undo() (cx, cy int, ok bool) {
	if m := g.Pop(); m != nil {
		g.Screen[m.Y1][m.X1] = m.D1
		g.Screen[m.Y2][m.X2] = m.D2
		g.Count++
		g.Removed--
		if g.Seq > 0 {
			g.Seq--
		}
		return m.X2, m.Y2, true
	}

	return -1, -1, false
}

var WinBanner = [][]Dir{
	{Down, Down, Up, Up, Down, Down, Up, Up, Down, Down, Down, Down, Up, Up, Down, Down, Up, Up, Down, Down},
	{Up, Up, Up, Up, Up, Up, Up, Down, Up, Up, Up, Up, Down, Up, Up, Up, Up, Up, Up, Up},
	{Up, Up, Up, Up, Up, Up, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Down, Up, Down, Down, Up, Empty, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Up, Up, Empty, Empty, Up, Up, Up, Up, Up, Up, Up, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Up, Up, Down, Down, Up, Up, Up, Up, Up, Down, Down, Up, Up},
	{Left, Left, Up, Up, Empty, Empty, Up, Empty, Up, Up, Up, Up, Empty, Empty, Empty, Up, Up, Up, Up, Right},
	{Left, Left, Left, Left, Left, Left, Left, Left, Left, Left, Right, Right, Right, Right, Right, Right, Right, Right, Right, Right},
	{Down, Down, Down, Empty, Empty, Empty, Down, Down, Down, Down, Down, Down, Down, Down, Empty, Empty, Empty, Down, Down, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Empty, Empty, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Down, Empty, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Empty, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Up, Down, Empty, Up, Up, Right},
	{Down, Up, Up, Empty, Down, Empty, Up, Up, Down, Up, Up, Down, Up, Up, Down, Up, Down, Up, Up, Right},
	{Down, Up, Up, Down, Up, Down, Up, Up, Down, Up, Up, Down, Up, Up, Down, Down, Up, Up, Up, Right},
	{Down, Down, Up, Up, Down, Up, Up, Down, Down, Up, Up, Down, Up, Up, Down, Down, Down, Up, Up, Right},
}

func (g *Game) Winner() bool {
	ww, hw := len(WinBanner[0]), len(WinBanner)

	if ww >= g.Width || hw >= g.Height {
		return false
	}

	ww = (g.Width - ww) / 2
	hw = (g.Height - hw) / 2

	g.Count = 0

	for y, row := range WinBanner {
		for x, col := range row {
			g.Screen[hw+y][ww+x] = col
			if col != Empty {
				g.Count++
			}
		}
	}

	return true
}

var game Game
