package configs

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

// promptRequired loops until a non-empty value is entered.
func promptRequired(label, hint string) string {
	for {
		if hint != "" {
			fmt.Printf("🔑 請輸入 %s\n   (%s)\n   > ", label, hint)
		} else {
			fmt.Printf("🔑 請輸入 %s\n   > ", label)
		}
		val, _ := stdinReader.ReadString('\n')
		val = strings.TrimSpace(val)
		if val != "" {
			return val
		}
		fmt.Printf("   ⚠  %s 不可為空，請重新輸入。\n\n", label)
	}
}

// promptOptional returns the user's input, or defaultVal if left empty.
func promptOptional(label, hint, defaultVal string) string {
	if hint != "" {
		fmt.Printf("💡 請輸入 %s（可選，直接 Enter 略過）\n   (%s)\n   > ", label, hint)
	} else {
		fmt.Printf("💡 請輸入 %s（可選，直接 Enter 略過）\n   > ", label)
	}
	val, _ := stdinReader.ReadString('\n')
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	return val
}
