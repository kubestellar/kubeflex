package util

import (
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	colorWaiting = color.FgCyan
)

// PrintWithIcon prints a message with defined colors and a spinning icon in front
// that can be changed to checkmark when completed.
func PrintStatus(message string, done chan bool, wg *sync.WaitGroup) {
	// Call the utility function in a go routine with a sample message and color
	wg.Add(1)
	go printWithIcon(message, colorWaiting, done, wg)
}

func printWithIcon(message string, colorAttr color.Attribute, done chan bool, wg *sync.WaitGroup) {
	// Define the icons for waiting and done states
	//waitingIcons := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆"}
	waitingIcons := []string{"◐", "◓", "◑", "◒"}
	//waitingIcons := []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}

	//waitingIcons := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	doneIcon := "✔"

	// Create a color printer with the first color attribute
	printer := color.New(colorAttr)

	// Loop until the done channel receives a value
	for {
		select {
		case <-done:
			// Clear the previous icon and message
			printer.Printf("\r   \r")
			// Print the message with the done icon in green
			color.Green("%s %s\n", doneIcon, message)
			wg.Done()
			return
		default:
			// Print the message with each waiting icon in sequence
			for _, icon := range waitingIcons {
				printer.Printf("\r%s %s", icon, message)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}
