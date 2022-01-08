package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
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

type Sequence struct {
	list   []int
	lindex int
	maxval int
}

func (s *Sequence) Len() int {
	return len(s.list)
}

func (s *Sequence) Reset(add bool) {
	s.lindex = 0
	if add {
		s.list = append(s.list, rand.Intn(s.maxval))
	}
}

func (s *Sequence) Next() int {
	if s.lindex == len(s.list) {
		return -1
	}

	curr := s.list[s.lindex]
	s.lindex++
	return curr
}

func (s *Sequence) HasNext() bool {
	next := s.lindex < len(s.list)
	return next
}

var (
	ww = 800
	wh = 800

	playInterval = 500 * time.Millisecond
	resetTime    = time.Second

	pads = []Pad{
		{new(widget.Clickable), "1", color.NRGBA{A: 255, R: 0, G: 200, B: 0}},   // green
		{new(widget.Clickable), "2", color.NRGBA{A: 255, R: 255, G: 0, B: 0}},   // red
		{new(widget.Clickable), "3", color.NRGBA{A: 255, R: 255, G: 255, B: 0}}, // yellow
		{new(widget.Clickable), "4", color.NRGBA{A: 255, R: 0, G: 128, B: 255}}, // blue
	}

	sequence = Sequence{maxval: 4}
)

func main() {
	flag.IntVar(&ww, "width", ww, "initial window width")
	flag.IntVar(&wh, "height", wh, "initial window height")
	center := flag.Bool("center", false, "center window")
	flag.Parse()

	rand.Seed(time.Now().Unix())

	audioInit()

	go func() {
		w := app.NewWindow(
			app.Title("Simon"),
			app.Size(unit.Px(float32(ww)), unit.Px(float32(wh))),
			//app.MinSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
			//app.MaxSize(unit.Px(float32(ww)), unit.Px(float32(wh))),
		)

		if *center {
			w.Center()
		}

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

	// grid := outlay.Grid{Num: 2, Axis: layout.Horizontal}
	simonPlay := true
	terminating := false
	selected := -1

	log.Println("simon play...")
	sequence.Reset(true)

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)
			if simonPlay || terminating {
				gtx = gtx.Disabled()
			}

			if simonPlay { // the FrameEvent is from invalidate
				simon := sequence.Next()
				if simon >= 0 {
					selected = simon
					time.AfterFunc(playInterval, w.Invalidate)
				} else {
					log.Println("user play...")
					simonPlay = false
					sequence.Reset(false)
				}
			}

			user := -1

			bbutton := func(gtx layout.Context, index int) layout.Widget {
				return func(gtx layout.Context) layout.Dimensions {
					pad := pads[index]

					if !simonPlay {
						if selected >= 0 { // from keyboard
							user = selected
						} else {
							for _, ev := range gtx.Events(pad.label) {
								ev, _ := ev.(pointer.Event)

								if ev.Type == pointer.Press {
									user = index
								} else if ev.Type == pointer.Release {
									user = -1
								}
							}

							selected = user
						}
					}

					if selected == index {
						log.Println("play", index+1)
						audioPlay(selected)
					}

					return pad.Layout(gtx, th, selected == index)
				}
			}

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(0.5, bbutton(gtx, 0)),
						layout.Flexed(0.5, bbutton(gtx, 1)),
					)
				}),
				layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(0.5, bbutton(gtx, 2)),
						layout.Flexed(0.5, bbutton(gtx, 3)),
					)
				}),
			)

			if !simonPlay && user >= 0 {
				// there are FrameEvents that are not from button clicks
				// if user >= 0 it was a button click
				simon := sequence.Next()
				if simon >= 0 {
					log.Println("simon", simon+1, "user", user+1)
					if simon != user {
						terminating = true

						time.AfterFunc(playInterval, func() {
							log.Println("Longest correct sequence:", sequence.Len()-1)
							audioPlay(audioBuzz)

							time.AfterFunc(resetTime, func() {
								w.Close()
							})
						})
					}
				}
				if !terminating && !sequence.HasNext() {
					time.AfterFunc(resetTime, func() {
						log.Println("simon play...")
						simonPlay = true
						sequence.Reset(true)
						w.Invalidate()
					})
				}
			}

			for i := 0; i < len(pads); i++ {
				var pos image.Rectangle

				switch i {
				case 0:
					pos = image.Rect(0, 0, e.Size.X/2, e.Size.Y/2)
				case 1:
					pos = image.Rect(e.Size.X/2, 0, e.Size.X, e.Size.Y/2)
				case 2:
					pos = image.Rect(0, e.Size.Y/2, e.Size.X/2, e.Size.Y)
				case 3:
					pos = image.Rect(e.Size.X/2, e.Size.Y/2, e.Size.X, e.Size.Y)
				}

				// Register to listen for pointer events.
				pr := clip.Rect(pos).Push(gtx.Ops)
				pointer.InputOp{Tag: pads[i].label, Types: pointer.Press | pointer.Release}.Add(gtx.Ops)
				pr.Pop()
			}

			e.Frame(gtx.Ops)
			selected = -1

		case key.Event:
			if e.State == key.Press {
				switch e.Name {
				case "1", "2", "3", "4":
					if !simonPlay {
						selected = int(e.Name[0] - '1')
						w.Invalidate()
					}

				case "X", "Q":
					w.Close()
				}
			} else {
				selected = -1
				w.Invalidate()
			}
		}
	}
}
