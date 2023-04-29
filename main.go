package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jim-at-jibba/telecharger/data"
	"github.com/jim-at-jibba/telecharger/tui"
)

func main() {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := fmt.Sprintf("%s/.config/telecharger", dirname)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err := os.Mkdir(configPath, os.ModePerm)
		if err != nil {
			log.Print(err.Error())
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s/app.env", configPath)); os.IsNotExist(err) {
		f, err := os.Create(fmt.Sprintf("%s/app.env", configPath))
		if err != nil {
			log.Fatal(err)
		}

		_, err2 := f.WriteString("DOWNLOAD_PATH=.")

		if err2 != nil {
			log.Fatal(err2)
		}
		defer f.Close()
	}

	data.OpenDatabase()
	data.CreateQueueTable()
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	tui.Models = []tea.Model{tui.InitialModel(), tui.NewForm()}
	m := tui.Models[tui.Info]
	tui.P = tea.NewProgram(m)

	if _, err := tui.P.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
