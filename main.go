package main

import (
	"fmt"
	"image"
	"math"

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

// ---- 2. 時間バー（待ち時間）を画像として描く ----
// 四角1個 = サービス1回分(Ts)。待ち Tw を「Ts 何個分か」で並べる。

func renderBar(lambda, ts float64) image.Image {
	const CW, CH = 560, 160
	dc := gg.NewContext(CW, CH)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	m := metrics(lambda, ts)

	x0, y, h, u := 20.0, 60.0, 44.0, 52.0 // u = 四角1個の幅(=Ts)
	maxUnits := 8.0                       // 表示できる最大個数

	if !m.Stable {
		dc.SetRGB(0.85, 0.2, 0.2)
		dc.DrawRectangle(x0, y, CW-2*x0, h)
		dc.Fill()
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored("UNSTABLE (rho >= 1): queue grows forever",
			CW/2, y+h/2, 0.5, 0.5)
		return dc.Image()
	}

	waitUnits := m.Rho / (1 - m.Rho) // = Tw / Ts
	drawUnits := waitUnits
	overflow := false
	if drawUnits > maxUnits {
		drawUnits = maxUnits
		overflow = true
	}

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

	dc.SetRGB(0.2, 0.7, 0.3)
	dc.DrawRectangle(xEnd, y, u, h)
	dc.Fill()

	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored("WAIT (Tw)", x0, y-10, 0, 0.5)
	dc.DrawStringAnchored("SERVICE (Ts)", xEnd, y-10, 0, 0.5)
	dc.DrawString(fmt.Sprintf(
		"Tw = %.2f  =  %.1f x Ts     Ts = %.2f     total W = %.2f     rho = %.2f",
		m.Tw, waitUnits, m.Ts, m.W, m.Rho), x0, y+h+24)
	dc.DrawString("each square = one service time (Ts)", x0, y+h+44)

	return dc.Image()
}

// ---- 3. ポアソン分布（到着の M）を棒グラフで描く ----
// P(k) = (λt)^k / k! * e^-(λt)。時間 t の間に k 件来る確率。

func renderPoisson(lambda, t float64) image.Image {
	const CW, CH = 560, 280
	dc := gg.NewContext(CW, CH)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	mean := lambda * t // λt = 平均到着数

	// 表示する k の上限（平均+ばらつき分をカバー）
	kMax := int(mean+4*math.Sqrt(mean)) + 1
	if kMax < 8 {
		kMax = 8
	}
	if kMax > 20 {
		kMax = 20
	}

	// P(k) を漸化式で計算：P(0)=e^-mean, P(k)=P(k-1)*mean/k
	probs := make([]float64, kMax+1)
	p := math.Exp(-mean)
	probs[0] = p
	maxP := p
	for k := 1; k <= kMax; k++ {
		p = p * mean / float64(k)
		probs[k] = p
		if p > maxP {
			maxP = p
		}
	}

	// 描画領域
	left, right, top, bottom := 40.0, 20.0, 40.0, 34.0
	plotW := CW - left - right
	plotH := CH - top - bottom
	baseY := top + plotH
	bw := plotW / float64(kMax+1) // 1本あたりの幅

	// 棒と k ラベル
	for k := 0; k <= kMax; k++ {
		barH := 0.0
		if maxP > 0 {
			barH = probs[k] / maxP * plotH
		}
		x := left + float64(k)*bw
		dc.SetRGB(0.2, 0.5, 0.9)
		dc.DrawRectangle(x+bw*0.15, baseY-barH, bw*0.7, barH)
		dc.Fill()
		dc.SetRGB(0, 0, 0)
		dc.DrawStringAnchored(fmt.Sprintf("%d", k), x+bw*0.5, baseY+14, 0.5, 0.5)
	}

	// 軸と見出し
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawLine(left, baseY, left+plotW, baseY)
	dc.Stroke()
	dc.DrawString(fmt.Sprintf(
		"Poisson P(k): arrivals in time t     mean = lambda*t = %.2f", mean),
		left, 22)
	dc.DrawStringAnchored("k (number of arrivals)", CW/2, CH-8, 0.5, 0.5)

	return dc.Image()
}

// ---- 4. GUI 組み立て（2タブ + 共通 λ スライダー）----

func main() {
	a := app.New()
	w := a.NewWindow("M/M/1 Visualizer")

	// 共通スライダー：λ（到着率）
	lambda := widget.NewSlider(0.1, 8.0)
	lambda.Step = 0.1
	lambda.Value = 3.0

	// --- Waiting Time タブ ---
	waitView := canvas.NewImageFromImage(nil)
	waitView.FillMode = canvas.ImageFillContain
	waitView.SetMinSize(fyne.NewSize(560, 160))
	waitInfo := widget.NewLabel("")
	waitFormula := widget.NewLabel(
		"rho = lambda / mu = lambda * Ts   ( mu = 1/Ts )\n" +
			"Tw = rho/(1-rho) * Ts        W = Tw + Ts")

	ts := widget.NewSlider(0.05, 1.0)
	ts.Step = 0.05
	ts.Value = 0.2

	// --- Poisson タブ ---
	poisView := canvas.NewImageFromImage(nil)
	poisView.FillMode = canvas.ImageFillContain
	poisView.SetMinSize(fyne.NewSize(560, 280))
	poisInfo := widget.NewLabel("")
	poisFormula := widget.NewLabel(
		"P(k) = (lambda*t)^k / k! * e^-(lambda*t)")

	tObs := widget.NewSlider(0.1, 3.0)
	tObs.Step = 0.1
	tObs.Value = 1.0

	// どのスライダーが動いても両タブを再計算
	refresh := func() {
		l := lambda.Value

		// Waiting Time
		m := metrics(l, ts.Value)
		waitView.Image = renderBar(l, ts.Value)
		waitView.Refresh()
		if m.Stable {
			waitInfo.SetText(fmt.Sprintf("lambda=%.1f  Ts=%.2f  ->  rho=%.2f,  Tw=%.2f,  W=%.2f",
				l, ts.Value, m.Rho, m.Tw, m.W))
		} else {
			waitInfo.SetText(fmt.Sprintf("lambda=%.1f  Ts=%.2f  ->  rho=%.2f  UNSTABLE",
				l, ts.Value, m.Rho))
		}

		// Poisson
		mean := l * tObs.Value
		poisView.Image = renderPoisson(l, tObs.Value)
		poisView.Refresh()
		poisInfo.SetText(fmt.Sprintf("lambda=%.1f  t=%.1f  ->  mean=%.2f,  P(0)=%.3f (no arrival)",
			l, tObs.Value, mean, math.Exp(-mean)))
	}
	lambda.OnChanged = func(float64) { refresh() }
	ts.OnChanged = func(float64) { refresh() }
	tObs.OnChanged = func(float64) { refresh() }

	waitTab := container.NewVBox(
		waitView,
		widget.NewLabel("Ts (mean service time)"), ts,
		waitInfo,
		widget.NewSeparator(),
		waitFormula,
	)
	poisTab := container.NewVBox(
		poisView,
		widget.NewLabel("t (observation time)"), tObs,
		poisInfo,
		widget.NewSeparator(),
		poisFormula,
	)
	tabs := container.NewAppTabs(
		container.NewTabItem("Waiting Time", waitTab),
		container.NewTabItem("Poisson", poisTab),
	)

	// 共通 λ スライダーは下部に固定
	lambdaBox := container.NewVBox(widget.NewLabel("lambda (arrival rate) -- shared"), lambda)
	w.SetContent(container.NewBorder(nil, lambdaBox, nil, nil, tabs))
	w.Resize(fyne.NewSize(600, 560))

	refresh()
	w.ShowAndRun()
}
