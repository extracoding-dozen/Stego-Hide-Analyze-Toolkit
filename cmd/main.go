package main

import (
	"Stego-Hide-Analyze-Toolkit/external/ui"
	"gioui.org/app"
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
