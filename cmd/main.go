package main

import (
	"gioui.org/app"
	"log"
	"os"
	"stego-hide-analize-toolkit/external/ui"
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
