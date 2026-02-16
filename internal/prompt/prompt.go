package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts user for Y/n confirmation.
// Returns false (No) by default when user presses Enter or inputs nothing.
// Returns true only when user explicitly inputs 'y' or 'Y'.
func Confirm(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// デフォルト: No (安全性重視)
	if input == "y" {
		return true, nil
	}

	return false, nil
}
