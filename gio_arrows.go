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

	_ "embed"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/disintegration/imaging"
)

var (
	//go:embed assets/up-arrow.png
	pngUp []byte

	bgColor = color.NRGBA{0, 0, 32, 255}

	dirs [5]image.Image
	cell image.Point

	canvas draw.Image

	width  = 20
	height = 20

	game Game
)

func main() {
	flag.IntVar(&width, "width", width, "screen width")
	flag.IntVar(&height, "height", height, "screen height")
	audio := flag.Bool("audio", true, "play audio effects")
	flag.Parse()

	if width <= 0 || height <= 0 {
		log.Fatal("invalid width or height")
	}

	width += 2  // add border
	height += 2 // to simplify boundary checks

	// Initialize audio
	if *audio {
		audioInit()
	}

	if img, err := png.Decode(bytes.NewBuffer(pngUp)); err != nil {
		log.Fatal(err)
	} else {
		cell = img.Bounds().Size()

		dirs[Empty] = imaging.New(cell.X, cell.Y, bgColor)
		dirs[Up] = img
		dirs[Left] = imaging.Rotate90(dirs[Up])
		dirs[Down] = imaging.Rotate90(dirs[Left])
		dirs[Right] = imaging.Rotate90(dirs[Down])
	}

	game.Setup(width, height, cell.X, cell.Y)

	go func() {
		w := app.NewWindow(app.Size(unit.Px(float32(width*cell.X)), unit.Px(float32(height*cell.Y))))
		loop(w)
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) {
	// th := material.NewTheme(gofont.Collection())
	var ops op.Ops

	for e := range w.Events() {
		switch e := e.(type) {
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			render(gtx, e.Size)
			e.Frame(gtx.Ops)
		case system.DestroyEvent:
			if e.Err != nil {
				fmt.Println(e.Err)
			}
			return
		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case key.NameEscape, "Q", "X":
					w.Close()

				case key.NameSpace:
					w.Invalidate()

				case "R": // reset
					audioPlay(Undo)
					game.Setup(width, height, cell.X, cell.Y)
					w.Invalidate()

				case "S": // reshuffle
					audioPlay(Shuffle)
					game.Shuffle()
					w.Invalidate()

				case "H": // help: remove all "free" arrows
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
							// s.PostEvent(tcell.NewEventInterrupt(true))
						}
					} else if moved != None {
						game.Seq = 0
					}

					w.Invalidate()
				}
			}
		}
	}
}

func render(gtx layout.Context, sz image.Point) {
	px := gtx.Metric.Px(unit.Dp(float32(sz.X)))
	py := gtx.Metric.Px(unit.Dp(float32(sz.Y)))

	if canvas == nil || canvas.Bounds().Size().X != px || canvas.Bounds().Size().Y != py {
		canvas = imaging.New(px, py, bgColor)
	} else {
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.ZP, draw.Src)
	}

	for y, row := range game.Screen {
		for x, col := range row {
			im := dirs[col]

			draw.Draw(canvas,
				im.Bounds().Add(image.Point{x * cell.X, y * cell.Y}),
				im, image.Point{}, draw.Over)
		}
	}

	canvasOp := paint.NewImageOp(canvas)
	img := widget.Image{Src: canvasOp}
	img.Scale = float32(sz.X) / float32(gtx.Px(unit.Dp(float32(sz.X))))
	img.Layout(gtx)
}
