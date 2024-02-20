//go:generate fyne bundle -name SOUND_FILE -o sounddata.go sounds/timer.mp3
package main

const APP_NAME = "Pomodorik"

const APP_WIDTH = 250
const APP_HEIGHT = 250

// pause name: seconds
var DEFAULT_TIMERS = map[string]int{ 
	"TOMATO": 1200, // 1200 sec = 20 min
	"SHORT": 300,
	"LONG": 600,
}
