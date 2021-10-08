package main

import (
	"math/rand"
	"time"
)

// Arrow directions
type Dir int8

type Updates int8

const (
	InvalidDir = Dir(-1)
	Empty      = Dir(0)
	Up         = Dir(1)
	Down       = Dir(2)
	Left       = Dir(3)
	Right      = Dir(4)

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
	MaxSeq  int

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

	g.simplify()
}

//
// shuffle arrows
// (actually replace/rotate arrows where present)
//
func (g *Game) Shuffle(dir Dir) {
	g.Count = 0
	g.Seq = 0

	for y, row := range g.Screen {
		for x, col := range row {
			if col != Empty {
				g.Count++

				switch dir {
				case Up, Left:
					// rotate left
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

				case Down, Right:
					// rotate right
					switch g.Screen[y][x] {
					case Up:
						g.Screen[y][x] = Right
					case Down:
						g.Screen[y][x] = Left
					case Left:
						g.Screen[y][x] = Up
					case Right:
						g.Screen[y][x] = Down
					}

				default:
					// random shuffle
					var newdir Dir
					for newdir = g.Screen[y][x]; newdir == g.Screen[y][x]; newdir = Dir(rand.Intn(DirCount) + 1) { // 0 is Empty
						// try again
					}

					g.Screen[y][x] = newdir
				}
			}
		}
	}

	if dir == Empty {
		g.simplify()
	}

	g.stack = g.stack[:0]
}

func (g *Game) simplify() {
	opposite := map[Dir]Dir{
		Up:    Down,
		Down:  Up,
		Left:  Right,
		Right: Left,
	}

	for y, row := range g.Screen {
		for x, col := range row {
			if col == Empty {
				continue
			}

			for c := 0; c < DirCount; c++ {
				opp := opposite[col]

				if g.Screen[y][x-1] == opp || g.Screen[y][x+1] == opp || g.Screen[y-1][x] == opp || g.Screen[y+1][x] == opp {
					switch col {
					case Up:
						col = Left
					case Down:
						col = Right
					case Left:
						col = Down
					case Right:
						col = Up
					}

					continue
				}

				break
			}

			g.Screen[y][x] = col

		}
	}
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

func (g *Game) Peek(x, y int) (int, int, Dir) {
	if cx, cy, ok := g.Coords(x, y); ok {
		return cx, cy, g.Screen[cy][cx]
	}

	return -1, -1, InvalidDir
}

func (g *Game) canRemove(cur Dir, x, y int) bool {
	cell := g.Screen[y][x]
	return /* cell == cur || */ cell == Empty
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

		curdir := g.Screen[cy][cx]

		update := func(removing bool, x, y int) (ret Updates) {
			if removing { // got to the end, remove current arrow
				g.Count--
				g.Removed++
				g.Seq++
				if g.Seq > g.MaxSeq {
					g.MaxSeq = g.Seq
				}
				ret = Remove
			} else { // partial move
				if op != Move {
					return None // we requested full move
				}

				g.Screen[y][x] = curdir // move into new position
				ret = Move
			}

			g.Push(x, y, Empty, cx, cy, g.Screen[cy][cx])
			g.Screen[cy][cx] = Empty // remove from old position
			g.Moves++
			return
		}

		switch curdir {
		case Up:
			for py = cy; py > 0 && g.canRemove(Up, cx, py-1); py-- {
			}

			if py == cy {
				return
			}

			res = update(py == 0, cx, py)

		case Down:
			for py = cy; py < g.Height-1 && g.canRemove(Down, cx, py+1); py++ {
			}

			if py == cy {
				return
			}

			res = update(py == g.Height-1, cx, py)

		case Left:
			for px = cx; px > 0 && g.canRemove(Left, px-1, cy); px-- {
			}

			if px == cx {
				return
			}

			res = update(px == 0, px, cy)

		case Right:
			for px = cx; px < g.Width-1 && g.canRemove(Right, px+1, cy); px++ {
			}

			if px == cx {
				return
			}

			res = update(px == g.Width-1, px, cy)
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
			if g.Seq == g.MaxSeq {
				g.MaxSeq--
			}
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
