package main

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"fyne.io/systray"

	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/effects"

	"pomadorik/icon"
)

var TextColors = map[string]color.NRGBA{
	"green":      {85, 165, 34, 255},
	"grey":       {82, 82, 82, 255},
	"white":      {255, 255, 255, 255},
	"lightgrey":  {57, 57, 57, 255},
	"lightgrey2": {142, 142, 142, 255},
}

var TIMER = DEFAULT_TIMERS["TOMATO"] 
var TICKER *time.Ticker = nil
var TIMER_TXT *canvas.Text = nil 

type BtnHandlerFn func(string, *canvas.Text) func()
var mainWindow fyne.Window
var App fyne.App 

func main() {
	App = app.NewWithID(APP_NAME)

	// setup window
	mainWindow = App.NewWindow(APP_NAME)
	mainWindow.Resize(fyne.NewSize(APP_WIDTH, APP_HEIGHT))
	mainWindow.SetIcon(icon.Data)

	if desk, ok := App.(desktop.App); ok {
		setupSystray(desk)
	}

	// setting intercept not to close app, but hide window,
	// and close only via tray
	mainWindow.SetCloseIntercept(func() {
		mainWindow.Hide()
	})

	content := buildContent(func (timerName string, timerTxt *canvas.Text) func() {
		TIMER_TXT = timerTxt

		// set on "space" start a tomato timer
		// https://developer.fyne.io/api/v1.4/keyname.html
		mainWindow.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
			switch k.Name {
			case fyne.KeySpace:
				startCountdown(DEFAULT_TIMERS["TOMATO"])
			}
		})

		return func() {
			startCountdown(DEFAULT_TIMERS[timerName])
		}
	})

	mainWindow.SetContent(content)
	fmt.Println("window init...")

	mainWindow.Show()
	App.Lifecycle().SetOnStarted(func() {
		systray.SetTooltip(APP_NAME)
		systray.SetTitle(APP_NAME)
	})
	App.Run()
}

func setupSystray(desk desktop.App) {
	// Set up menu
	desk.SetSystemTrayIcon(theme.NewThemedResource(icon.Disabled))

	menu := fyne.NewMenu(APP_NAME,
		fyne.NewMenuItem("Open", mainWindow.Show),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Focus", func() {
			startCountdown(DEFAULT_TIMERS["TOMATO"])
		}),
		fyne.NewMenuItem("Short break", func() {
			startCountdown(DEFAULT_TIMERS["SHORT"])
		}),
		fyne.NewMenuItem("Long break", func() {
			startCountdown(DEFAULT_TIMERS["LONG"])
		}),
		)
	desk.SetSystemTrayMenu(menu)
}

func buildContent(onBtnHandler BtnHandlerFn) fyne.CanvasObject {
	greenColor := TextColors["green"]

	timer := buildTxtWithStyle(formatTimer(TIMER), greenColor, 40)
	tomatoBtn := widget.NewButton("Focus", onBtnHandler("TOMATO", timer))
	shortBrakeBtn := widget.NewButton("Short brake", onBtnHandler("SHORT", timer))
	longBrakeBtn := widget.NewButton("Long brake", onBtnHandler("LONG", timer))

	content := fyne.NewContainerWithLayout(
		layout.NewVBoxLayout(),

		// header (timer)
		container.New(layout.NewCenterLayout(), timer),

		// btns 
		tomatoBtn,
		buildSpace(),

		shortBrakeBtn,
		longBrakeBtn,
		buildSpace(),

		container.New(
			layout.NewCenterLayout(), 
			// container.New(layout.NewVBoxLayout(),
				buildTxtWithStyle(
					"Press \"Space\" to start Focus",
					TextColors["grey"],
					10,
				),
			// ),

		),
			
	)
	return content
}

func buildTxtWithStyle(title string, textColor color.NRGBA, textSize float32) *canvas.Text {
	txt := canvas.NewText(title, textColor)
	txt.TextSize = textSize
	// txt.Alignment = fyne.TextAlignTrailing 
	return txt
}

func buildLabelTxt(title string) *canvas.Text {
	txt := canvas.NewText(title, TextColors["grey"])
	txt.TextSize = 12
	return txt
}

func buildSpace() *canvas.Text {
	return buildLabelTxt("")
}

func updateTimerTxt(timer int, timerTxt *canvas.Text) {
	timerTxt.Text = formatTimer(timer) 
	timerTxt.Refresh()

	systray.SetTitle(fmt.Sprintf("%s (%s)", APP_NAME, timerTxt.Text))
	systray.SetTooltip(fmt.Sprintf("%s (%s)", APP_NAME, timerTxt.Text))
}

func startCountdown(defaultTime int) {
	fyne.CurrentApp().Driver().SetDisableScreenBlanking(true)
	if desk, ok := App.(desktop.App); ok {
		desk.SetSystemTrayIcon(icon.Data)
	}
	// if timer already started, at again start, just stop it
	TIMER = defaultTime
	updateTimerTxt(TIMER, TIMER_TXT) 

	if TICKER != nil {
		TICKER.Stop()
	}

	TICKER = startTimer(func (ticker *time.Ticker) {
		updateTimerTxt(TIMER, TIMER_TXT)

		if TIMER == 0 {
			playSound()
			ticker.Stop()
			TICKER = nil
			mainWindow.Show()
			mainWindow.RequestFocus()

			if desk, ok := App.(desktop.App); ok {
				desk.SetSystemTrayIcon(theme.NewThemedResource(icon.Disabled))
			}
			fyne.CurrentApp().Driver().SetDisableScreenBlanking(false)
		}

		TIMER--
	})
}

// https://gobyexample.com/tickers
func startTimer(onTickFn func(*time.Ticker)) *time.Ticker {
	ticker := time.NewTicker(1 * time.Second)
	done := make(chan bool)
	go func() {
		for {
			select {
				case <-done: return
				case <-ticker.C: 
					onTickFn(ticker)
			}
		}
	}()
	return ticker
}

func playSound() {
	stream, format, err := mp3.Decode(io.NopCloser(bytes.NewReader(SOUND_FILE.Content())))
	if err != nil {
		fyne.LogError("Unable to stream sound "+SOUND_FILE.Name(), err)
		return
	}

	volume := effects.Volume{ 
		Streamer: stream,
		Base: 2,
		Volume: 1.6,
		Silent: false,
	}

	// activate speakers 
	speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Second/10),
	)

	// play
	speaker.Play(&volume) 
}

func formatTimer(timer int) string {
	minutes := TIMER / 60
	seconds := TIMER % 60
	minZero := ""
	secZero := ""

	if minutes < 10 {
		minZero = "0"
	}
	if seconds < 10 {
		secZero = "0"
	}
	return fmt.Sprintf("%s%d:%s%d", minZero, minutes, secZero, seconds)
}
