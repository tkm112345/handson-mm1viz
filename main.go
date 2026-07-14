package main

import (
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"git.sr.ht/~sbinet/gg"
)

// ---- 1. M/M/1 の計算（Ts を入力にする）----

type Metrics struct {
	Rho    float64 // 利用率 ρ = λ·Ts
	Tw     float64 // 平均待ち時間
	Ts     float64 // 平均サービス時間（入力）
	W      float64 // 平均応答時間 = Tw + Ts
	Stable bool
}

func metrics(lambda, ts float64) Metrics {
	rho := lambda * ts // ρ = λ/μ = λ·Ts
	if rho >= 1 {
		return Metrics{Rho: rho, Ts: ts, Stable: false}
	}
	tw := rho / (1 - rho) * ts // Tw = ρ/(1-ρ)·Ts
	return Metrics{Rho: rho, Tw: tw, Ts: ts, W: tw + ts, Stable: true}
}

// ---- 2. 時間バーを画像として描く ----
// 四角1個 = サービス1回分(Ts)。待ち Tw を「Ts 何個分か」で並べる。

func renderBar(lambda, ts float64) image.Image {
	const CW, CH = 560, 160
	dc := gg.NewContext(CW, CH)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	m := metrics(lambda, ts)

	x0, y, h, u := 20.0, 60.0, 44.0, 52.0 // u = 四角1個の幅(=Ts)
	maxUnits := 8.0                       // 表示できる最大個数

	// 不安定なら赤帯で警告して終了
	if !m.Stable {
		dc.SetRGB(0.85, 0.2, 0.2)
		dc.DrawRectangle(x0, y, CW-2*x0, h)
		dc.Fill()
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored("UNSTABLE (rho >= 1): queue grows forever",
			CW/2, y+h/2, 0.5, 0.5)
		return dc.Image()
	}

	waitUnits := m.Rho / (1 - m.Rho) // = Tw / Ts（待ちは「サービス何個分」か）
	drawUnits := waitUnits
	overflow := false
	if drawUnits > maxUnits {
		drawUnits = maxUnits
		overflow = true
	}

	// 待ち時間（オレンジ）。四角ごとに白い区切り線で「■の並び」に見せる
	dc.SetRGB(0.95, 0.6, 0.15)
	dc.DrawRectangle(x0, y, drawUnits*u, h)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.SetLineWidth(1)
	for i := 1.0; i < drawUnits; i++ {
		dc.DrawLine(x0+i*u, y, x0+i*u, y+h)
		dc.Stroke()
	}
	xEnd := x0 + drawUnits*u
	if overflow {
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(">>", xEnd+6, y+h/2, 0, 0.5)
		xEnd += 22
	}

	// サービス（緑）= 四角1個
	dc.SetRGB(0.2, 0.7, 0.3)
	dc.DrawRectangle(xEnd, y, u, h)
	dc.Fill()

	// ラベル
	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored("WAIT (Tw)", x0, y-10, 0, 0.5)
	dc.DrawStringAnchored("SERVICE (Ts)", xEnd, y-10, 0, 0.5)
	dc.DrawString(fmt.Sprintf(
		"Tw = %.2f  =  %.1f x Ts     Ts = %.2f     total W = %.2f     rho = %.2f",
		m.Tw, waitUnits, m.Ts, m.W, m.Rho), x0, y+h+24)
	dc.DrawString("each square = one service time (Ts)", x0, y+h+44)

	return dc.Image()
}

// ---- 3. GUI 組み立て ----

func main() {
	a := app.New()
	w := a.NewWindow("M/M/1 Waiting Time Visualizer")

	view := canvas.NewImageFromImage(nil)
	view.FillMode = canvas.ImageFillContain
	view.SetMinSize(fyne.NewSize(560, 160))

	info := widget.NewLabel("")

	// 数式の補足（常時表示）
	formula := widget.NewLabel(
		"rho = lambda / mu = lambda * Ts   ( mu = 1/Ts )\n" +
			"Tw = rho/(1-rho) * Ts        W = Tw + Ts")

	lambda := widget.NewSlider(0.1, 8.0)
	lambda.Step = 0.1
	lambda.Value = 3.0

	ts := widget.NewSlider(0.05, 1.0)
	ts.Step = 0.05
	ts.Value = 0.2

	refresh := func() {
		l, s := lambda.Value, ts.Value
		m := metrics(l, s)
		view.Image = renderBar(l, s)
		view.Refresh()
		if m.Stable {
			info.SetText(fmt.Sprintf("lambda=%.1f  Ts=%.2f  ->  rho=%.2f,  Tw=%.2f,  W=%.2f",
				l, s, m.Rho, m.Tw, m.W))
		} else {
			info.SetText(fmt.Sprintf("lambda=%.1f  Ts=%.2f  ->  rho=%.2f  UNSTABLE",
				l, s, m.Rho))
		}
	}
	lambda.OnChanged = func(float64) { refresh() }
	ts.OnChanged = func(float64) { refresh() }

	form := container.NewVBox(
		widget.NewLabel("lambda (arrival rate)"), lambda,
		widget.NewLabel("Ts (mean service time)"), ts,
		info,
		widget.NewSeparator(),
		formula,
	)
	w.SetContent(container.NewBorder(nil, form, nil, nil, view))
	w.Resize(fyne.NewSize(600, 420))

	refresh()
	w.ShowAndRun()
}
