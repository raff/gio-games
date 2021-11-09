package main

import (
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app" // app contains Window handling.
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key" // key is used for keyboard events.
	"gioui.org/io/pointer"
	"gioui.org/io/system" // system is used for system events (e.g. closing the window).
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gioui.org/x/outlay"
)

type (
	D = layout.Dimensions
	C = layout.Context
)

type Pad struct {
	button *widget.Clickable
	label  string
	color  color.NRGBA
}

func (pad Pad) Layout(gtx C, th *material.Theme, active bool) D {
	return material.Clickable(gtx, pad.button, func(gtx C) D {
		return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
			c := pad.color
			if !active {
				c = Darker(c)
			}

			dims := DrawRect(gtx, c, gtx.Constraints.Max, 20)
			layout.Center.Layout(gtx, material.H1(th, pad.label).Layout)
			return dims
		})
	})
}

func DrawRect(gtx C, background color.NRGBA, size image.Point, radii float32) D {
	bounds := f32.Rectangle{Max: f32.Pt(float32(size.X), float32(size.Y))}
	paint.FillShape(gtx.Ops, background, clip.UniformRRect(bounds, radii).Op(gtx.Ops))
	return layout.Dimensions{Size: size}
}

// Brighter blends color towards a brighter color.
func Brighter(c color.NRGBA) (d color.NRGBA) {
	const r = 0x40 // lighten ratio
	return color.NRGBA{
		R: byte(255 - int(255-c.R)*(255-r)/256),
		G: byte(255 - int(255-c.G)*(255-r)/256),
		B: byte(255 - int(255-c.B)*(255-r)/256),
		A: c.A,
	}
}

// Darker blends color towards a darker color.
func Darker(c color.NRGBA) (d color.NRGBA) {
	const r = 2 // darken ration
	return color.NRGBA{
		R: c.R / r,
		G: c.G / r,
		B: c.B / r,
		A: c.A,
	}
}

var (
	ww = float32(800)
	wh = float32(600)
)

func main() {
	go func() {
		w := app.NewWindow(
			app.Title("Simon"),
			app.Size(unit.Px(ww), unit.Px(wh)),
			app.MinSize(unit.Px(ww), unit.Px(wh)),
			app.MaxSize(unit.Px(ww), unit.Px(wh)),
		)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	var ops op.Ops

	th := material.NewTheme(gofont.Collection())

	pads := []Pad{
		{new(widget.Clickable), "1", color.NRGBA{A: 255, R: 0, G: 200, B: 0}},   // green
		{new(widget.Clickable), "2", color.NRGBA{A: 255, R: 255, G: 0, B: 0}},   // red
		{new(widget.Clickable), "3", color.NRGBA{A: 255, R: 255, G: 255, B: 0}}, // yellow
		{new(widget.Clickable), "4", color.NRGBA{A: 255, R: 0, G: 128, B: 255}}, // blue
	}

	// var seq []int

	grid := outlay.Grid{Num: 2, Axis: layout.Horizontal}
	disable := false
	pressed := -1

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if disable {
				gtx = gtx.Disabled()
			}

			grid.Layout(gtx, len(pads), func(gtx C, i int) D {
				gtx.Constraints.Max.X = gtx.Constraints.Max.X / 2
				gtx.Constraints.Max.Y = int(wh) / 2

				if pressed < 0 && pads[i].button.Pressed() {
					pressed = i
				}

				dims := pads[i].Layout(gtx, th, pressed == i)
				pointer.CursorNameOp{Name: pointer.CursorPointer}.Add(gtx.Ops)
				return dims
			})

			e.Frame(gtx.Ops)
			pressed = -1

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case "1", "2", "3", "4":
					if !disable {
						pressed = int(e.Name[0] - '1')
						w.Invalidate()
					}

				case "X", "Q":
					w.Close()

				case "`":
					disable = !disable
					w.Invalidate()
				}
			} else {
				pressed = -1
				w.Invalidate()
			}
		}
	}
}
