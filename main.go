package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jim-at-jibba/telecharger/data"
	"github.com/jim-at-jibba/telecharger/tui"
	util "github.com/jim-at-jibba/telecharger/utils"
)

func main() {
	cfg, err := util.ParseConfig()
	if err != nil {
		log.Println(err)
	}

	data.OpenDatabase()
	data.CreateQueueTable()

	if cfg.Settings.EnableLogging {
		f, err := tea.LogToFile("debug.log", "debug")
		log.Printf("In debug mode")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	tui.Models = []tea.Model{tui.InitialModel(cfg), tui.NewForm()}
	m := tui.Models[tui.Info]
	tui.P = tea.NewProgram(m)

	if _, err := tui.P.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
