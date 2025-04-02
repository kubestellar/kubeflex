/*
Copyright 2023 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
func PrintStatus(message string, done chan bool, wg *sync.WaitGroup, chattyStatus bool) {
	// Call the utility function in a go routine with a sample message and color
	wg.Add(1)
	go printWithIcon(message, colorWaiting, done, wg, chattyStatus)
}

func printWithIcon(message string, colorAttr color.Attribute, done chan bool, wg *sync.WaitGroup, chattyStatus bool) {
	// Define the icons for waiting and done states
	// there are different alternatives for the waiting icons
	//waitingIcons := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆"}
	//waitingIcons := []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	//waitingIcons := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	waitingIcons := []string{"◐", "◓", "◑", "◒"}
	doneIcon := "✔"

	// Create a color printer with the first color attribute
	printer := color.New(colorAttr)

	// Loop until the done channel receives a value
	firstTime := true
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
				if firstTime || chattyStatus {
					printer.Printf("\r%s %s", icon, message)
					firstTime = false
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}
