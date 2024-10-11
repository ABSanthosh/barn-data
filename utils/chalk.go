package utils

import "fmt"

func Chalk(format string, color string, args ...interface{}) {
	var Reset = "\033[0m"
	var Red = "\033[31m"
	var Green = "\033[32m"
	var Yellow = "\033[33m"
	var Blue = "\033[34m"
	var Magenta = "\033[35m"
	var Cyan = "\033[36m"
	var Gray = "\033[37m"
	var White = "\033[97m"

	// Format the text using fmt.Sprintf
	formattedText := fmt.Sprintf(format, args...)

	// Use a switch statement to set the color
	switch color {
	case "red":
		fmt.Printf(Red + formattedText + Reset)
	case "green":
		fmt.Printf(Green + formattedText + Reset)
	case "yellow":
		fmt.Printf(Yellow + formattedText + Reset)
	case "blue":
		fmt.Printf(Blue + formattedText + Reset)
	case "magenta":
		fmt.Printf(Magenta + formattedText + Reset)
	case "cyan":
		fmt.Printf(Cyan + formattedText + Reset)
	case "gray":
		fmt.Printf(Gray + formattedText + Reset)
	case "white":
		fmt.Printf(White + formattedText + Reset)
	default:
		fmt.Print(formattedText)
	}
}
