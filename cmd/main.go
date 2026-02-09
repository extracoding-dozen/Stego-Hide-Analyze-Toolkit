package main

import (
	"gioui.org/app"
	"github.com/extracoding-dozen/Stego-Hide-Analyze-Toolkit/external/ui"
	"log"
	"os"
)

func main() {
	controller := ui.NewController()
	go func() {
		if err := controller.Run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
