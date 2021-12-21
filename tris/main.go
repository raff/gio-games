package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"time"

	"math/rand"

	_ "embed"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/disintegration/imaging"
)

const (
	hcount = 10
	vcount = 4

	border = 8
)

var (
	//go:embed assets/tiles.png
	pngTiles []byte

	tiles []draw.Image

	borderColor = color.NRGBA{80, 80, 80, 255}

	canvas draw.Image

	tw, th int // game tile width, height
	gw, gh int // number of horizontal and vertical tiles in game
	ww, wh int // window width and height

	cards []int // gw * gh tiles, card indices

	mcount = 2
)

func initGame() {
	if len(tiles) == 0 {
		img, err := png.Decode(bytes.NewBuffer(pngTiles))
		if err != nil {
			log.Fatal(err)
		}

		isz := img.Bounds().Size()
		hsize := isz.X / hcount
		vsize := isz.Y / vcount

		tw = hsize / 2
		th = vsize / 2

		//card_loop:
		for v, y := 0, 0; v < vcount; v++ {
			for h, x := 0, 0; h < hcount; h++ {
				card := v*hcount + h

				tile := imaging.New(tw, th, borderColor)
				im := imaging.Crop(img, image.Rect(x, y, x+hsize, y+vsize))
				im = imaging.Resize(im, tw-border, th-border-border, imaging.Box)
				draw.Draw(tile,
					im.Bounds().Add(image.Point{border, border}),
					im, image.Point{}, draw.Over)
				tiles = append(tiles, tile)

				cards = append(cards, card)
				cards = append(cards, card)
				cards = append(cards, card)
				cards = append(cards, card)
				cards = append(cards, card)
				cards = append(cards, card)
				x += hsize
			}

			y += vsize
		}
	}

	rand.Shuffle(len(cards), func(i, j int) {
		if cards[i] != -1 && cards[j] != -1 {
			cards[i], cards[j] = cards[j], cards[i]
		}
	})

	gw, gh = factors(len(cards))
	ww, wh = gw*tw, (gh+1)*th/2

	canvas = imaging.New(ww, wh, borderColor)
	drawCards(nil)
}

func drawCards(revs map[int]bool) {
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{borderColor}, image.ZP, draw.Src)

	for ti, ci := range cards {
		x := ti % gw
		y := (ti / gw) % gh

		if ci < 0 {
			continue
		}

		im := tiles[ci]

		if revs != nil && revs[ti] {
			im = imaging.Invert(im)
		}

		draw.Draw(canvas,
			im.Bounds().Add(image.Point{x * tw, y * th / 2}),
			im, image.Point{}, draw.Over)
	}
}

func cardIndex(x, y int) int {
	x /= tw
	y /= (th / 2)

	log.Println("cardIndex", x, y)

	if x >= 0 && x < gw && y >= 0 && y < gh {
		ci := gameIndex(x, y)
		if y == gh-1 || cards[ci+gw] == -1 { // last valid card in a column
			if cards[ci] >= 0 {
				log.Println("card", ci)
				return ci
			}
		}
	}

	return -1
}

func factors(n int) (int, int) {
	for i := n - 1; i > 1; i-- {
		if n%i == 0 {
			m1 := i
			m2 := n / i

			if m2 >= m1 {
				return m2, m1
			}
		}
	}

	return n, 1
}

func main() {
	rand.Seed(time.Now().Unix())

	initGame()
	fmt.Println(factors(len(cards)))

	go func() {
		w := app.NewWindow(
			app.Title("Tris"),
			app.Size(unit.Px(float32(ww)), unit.Px(float32(wh))),
			app.MinSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
			app.MaxSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
		)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func gameState() draw.Image {
	return canvas
}

//func gameCoords(x, y int) (int, int) {
//      return x / tw, y / th
//}

func gameIndex(x, y int) int {
	return y*gw + x
}

func loop(w *app.Window) error {
	var ops op.Ops
	var frame draw.Image

	var matches map[int]bool
	match := -1

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()
				case "R":
					initGame()
					frame = nil
					match = -1
					w.Invalidate()
				}
			}

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if frame == nil {
				frame = gameState()
			}

			for _, ev := range gtx.Events("tris") {
				if ev, ok := ev.(pointer.Event); ok {
					if ev.Type == pointer.Press {
						ci := cardIndex(int(ev.Position.X), int(ev.Position.Y))
						if ci >= 0 {
							card := cards[ci]

							if match != card {
								match = card
								matches = map[int]bool{ci: true}
							} else {
								matches[ci] = true
							}

							if len(matches) == mcount {
								for k, _ := range matches {
									cards[k] = -1
								}

								matches = nil
								match = -1
							}

							drawCards(matches)
							w.Invalidate()
						}
					}
				}
			}

			canvasOp := paint.NewImageOp(frame)
			img := widget.Image{Src: canvasOp}
			img.Scale = 1 / float32(gtx.Px(unit.Dp(1)))
			img.Layout(gtx)

			pr := pointer.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
			pointer.InputOp{Tag: "tris", Types: pointer.Press}.Add(gtx.Ops)
			pr.Pop()

			e.Frame(gtx.Ops)
		}
	}
}
