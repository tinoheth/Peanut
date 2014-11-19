// Plotter
package peanut

import (
	"code.google.com/p/plotinum/plot"
	"code.google.com/p/plotinum/plotter"
	//"code.google.com/p/plotinum/plotutil"
	//"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	"image"
	"image/color"
	_ "image/png"
	//"math"
	//"time"
)

func powerPlot(data []FloatSample) image.Image {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.X.Label.Text = "Zeit"
	p.Y.Label.Text = "Y"
	p.Y.Min = 0.0

	line, err := plotter.NewLine(FloatSampleData(data))
	if err == nil {
		line.Color = color.RGBA{R: 255, A: 255}
		p.Add(line)
	}

	//p.Save(600, 600, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 1400, 1000))
	c := vgimg.PngCanvas{Canvas: vgimg.NewImage(img)}
	p.Draw(plot.MakeDrawArea(c))
	println("Done plotting")
	return img
}

func (fp FloatSampleData) Len() int {
	return len(fp)
}

func (fp FloatSampleData) XY(i int) (float64, float64) {
	e := fp[i]
	return float64(e.Time.Unix()), float64(e.Value)
}
