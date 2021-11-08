package main

import (
	"bytes"
	"flag"
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
)

var (
	//go:embed assets/tiles.png
	pngTiles []byte

	tiles []draw.Image

	cover       draw.Image
	coverColor  = color.NRGBA{0, 0, 32, 255}
	borderColor = color.NRGBA{200, 200, 255, 0}

	waitTurn = 300 * time.Millisecond
	waitGame = 5 * time.Second

	canvas draw.Image

	tw, th int // game tile width, height
	gw, gh int // number of horizontal and vertical tiles in game

	maxcards = hcount * vcount

	cards  []int  // gw * gh tiles, card indices
	states []bool // gw * gh tiles, card states

	moves   int
	matches int
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

	card_loop:
		for v, y := 0, 0; v < vcount; v++ {
			for h, x := 0, 0; h < hcount; h++ {
				card := v*hcount + h
				if card >= maxcards {
					break card_loop
				}

				im := imaging.Crop(img, image.Rect(x, y, x+hsize, y+vsize))
				im = imaging.Resize(im, hsize/2, vsize/2, imaging.Box)
				tiles = append(tiles, im)

				cards = append(cards, card)
				cards = append(cards, card)
				x += hsize
			}

			y += vsize
		}

		tw = hsize / 2
		th = vsize / 2
	}

	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	cover = imaging.New(tw, th, borderColor)
	inset := imaging.New(tw-8, th-8, coverColor)
	draw.Draw(cover,
		inset.Bounds().Add(image.Point{4, 4}),
		inset, image.Point{}, draw.Over)

	gw, gh = factors(len(cards))

	states = make([]bool, len(cards))
	canvas = imaging.New(tw*gw, th*gh, coverColor)

	for ti, ci := range cards {
		x := ti % gw
		y := (ti / gw) % gh

		im := tiles[ci]

		draw.Draw(canvas,
			im.Bounds().Add(image.Point{x * tw, y * th}),
			im, image.Point{}, draw.Over)
	}

	moves = 0
	matches = 0
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
	flag.IntVar(&maxcards, "cards", maxcards, "maximum number of cards")
	flag.DurationVar(&waitTurn, "turn", waitTurn, "wait before hiding cards (between turns)")
	flag.DurationVar(&waitGame, "game", waitGame, "wait before hiding cards (at game start)")
	audio := flag.Bool("audio", true, "play audio")
	flag.Parse()

	if maxcards > hcount*vcount {
		maxcards = hcount * vcount
	}

	rand.Seed(time.Now().Unix())

	initGame()

	if *audio {
		audioInit()
	}

	go func() {
		w := app.NewWindow(
			app.Title("Memory"),
			app.Size(unit.Px(float32(gw*tw)), unit.Px(float32(gh*th))),
			app.MinSize(unit.Px(float32(gw*tw)), unit.Px(float32(gh*th))),
			app.MaxSize(unit.Px(float32(gw*tw)), unit.Px(float32(gh*th))),
		)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func gameState() draw.Image {
	im := imaging.Clone(canvas)

	for ti, found := range states {
		x := ti % gw
		y := (ti / gw) % gh

		if !found {
			draw.Draw(im,
				cover.Bounds().Add(image.Point{x * tw, y * th}),
				cover, image.Point{}, draw.Over)
		}
	}

	return im
}

func gameCoords(x, y int) (int, int) {
	return x / tw, y / th
}

func gameIndex(x, y int) int {
	return y*gw + x
}

func loop(w *app.Window) error {
	var ops op.Ops
	var frame draw.Image

	deal := make([]int, 0, 2)
	newGame := true

	defer func() {
		fmt.Println(moves, "moves", matches, "matches")
	}()

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
					deal = deal[:0]
					newGame = true
				}
			}

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			update := false

			for _, ev := range gtx.Events("memory") {
				if !newGame {
					if ev, ok := ev.(pointer.Event); ok && ev.Type == pointer.Press {
						x, y := gameCoords(int(ev.Position.X), int(ev.Position.Y))
						si := gameIndex(x, y)

						if len(deal) < 2 && states[si] == false {
							states[si] = true
							deal = append(deal, si)

							if len(deal) == 2 {
								moves++
							}

							audioPlay(AudioFlip)
							frame = nil
							update = true
						}
					}
				}
			}

			if update && len(deal) == 2 {
				d1, d2 := deal[0], deal[1]

				if cards[d1] == cards[d2] {
					deal = deal[:0]
					matches++
				} else {
					time.AfterFunc(waitTurn, func() {
						states[d1] = false
						states[d2] = false
						deal = deal[:0]
						frame = nil
						w.Invalidate()
						audioPlay(AudioReset)
					})
				}
			}

			if newGame {
				frame = canvas

				time.AfterFunc(waitGame, func() {
					newGame = false
					frame = nil
					fmt.Println("invalidate")
					w.Invalidate()
				})
			}

			if frame == nil {
				frame = gameState()
			}

			canvasOp := paint.NewImageOp(frame)
			img := widget.Image{Src: canvasOp}
			img.Scale = 1 / float32(gtx.Px(unit.Dp(1)))
			img.Layout(gtx)

			// Register to listen for pointer events.
			pr := pointer.Rect(image.Rectangle{Max: e.Size}).Push(gtx.Ops)
			pointer.InputOp{Tag: "memory", Types: pointer.Press}.Add(gtx.Ops)
			pr.Pop()

			e.Frame(gtx.Ops)

			if matches == maxcards {
				w.Close()
			}
		}
	}
}
