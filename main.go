package main

import (
	"fmt"
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// ---- 1. M/M/1 の計算 ----

// MetricsはM/M/1の各指標をまとめた構造体
type Metrics struct {
	Rho float64 // 利用率 ρ
	Wq float64 // 平均待ち時間
	W float64 // 平均応答時間
	Stable bool // ρ<1 なら true
}

// metrics は到着率 lambda・サービス率 mu から指標を計算する
func metrics(lambda, mu float64) Metrics {
	rho := lambda / mu
	if rho >= 1 {
		return Metrics{Rho: rho, Stable: false}
	}
	ts := 1 / mu
	wq := rho / (1 - rho) * ts
	w := ts / ( 1 - rho)
	return Metrics{Rho: rho, Wq : wq, W: w, Stable: true}
}

// ---- 2. グラフを画像として生成 ----
// renderPlot は Wq-ρ 曲線と現在点を描き、image.Image で返す
func renderPlot(lambda, mu float64) image.Image {
	p := plot.New()
	p.Title.Text = "M/M/1 Queue: Wq vs ρ"
	p.X.Label.Text = "ρ (Utilization)"
	p.Y.Label.Text = "Wq (Average Waiting Time)"
	p.X.Min, p.X.Max = 0,1
	p.Y.Min = 0

	ts := 1 / mu

	// 曲線：ρ=0〜0.95 を刻んで Wq を計算
	pts := make(plotter.XYs, 0,96)
	for i := 0; i <= 95; i++ {
		rho := float64(i) / 100.0
		pts = append(pts, plotter.XY{X: rho, Y: rho / (1 - rho) * ts})
	}
	line, _ := plotter.NewLine(pts)
	line.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	line.Width = vg.Points(2)
	p.Add(line)

	// 現在点（安定時だけ赤丸で表示）
	m := metrics(lambda, mu)
	if m.Stable {
		cur, _ := plotter.NewScatter(plotter.XYs{{X: m.Rho, Y: m.Wq}})
		cur.GlyphStyle.Color = color.RGBA{R: 255, A: 255}
		cur.GlyphStyle.Radius = vg.Points(5)
		cur.GlyphStyle.Shape = draw.CircleGlyph{}
		p.Add(cur)
	}

	// plot → image.Image へ変換（ここが Fyne との橋渡し）
	c:= vgimg.New(vg.Points(500), vg.Points(300))
	p.Draw(draw.New(c))
	return c.Image()
}

// ---- 3. GUI 組み立て ----
func main() {
	a := app.New()
	w := a.NewWindow("M/M/1 Queue Visualizer")

	// グラフ表示用の画像部品
	graph := canvas.NewImageFromImage(nil)
	graph.FillMode = canvas.ImageFillContain
	graph.SetMinSize(fyne.NewSize(500, 300))

	// 数値表示用ラベル
	info := widget.NewLabel("")

	// スライダー2本
	lambda := widget.NewSlider(0.1,9.0)
	lambda.Step = 0.1
	lambda.Value = 3.0

	mu := widget.NewSlider(0.1,10.0)
	mu.Step = 0.1
	mu.Value = 5.0

	// 再計算
	refresh := func(){
		l,u := lambda.Value,mu.Value
		m := metrics(l,u)
		graph.Image = renderPlot(l,u)
		graph.Refresh()
		if m.Stable {
			info.SetText(fmt.Sprintf("λ=%.2f, μ=%.2f, ρ=%.2f, Wq=%.2f, W=%.2f", l,u,m.Rho,m.Wq,m.W))
		} else {
			info.SetText(fmt.Sprintf("λ=%.2f, μ=%.2f, ρ=%.2f (Unstable)", l,u,m.Rho))
		}
	}
	lambda.OnChanged = func(float64){refresh()}
	mu.OnChanged = func(float64){refresh()}

	// レイアウト：・中央にグラフ
	form := container.NewVBox(
		widget.NewLabel("lambda (arrival rate)"), lambda,
		widget.NewLabel("mu (arrival rate)"), mu,
		info,
	)
	w.SetContent(container.NewBorder(nil, form, nil, nil, graph))
	w.Resize(fyne.NewSize(560, 480))

	refresh()
	w.ShowAndRun()
}
