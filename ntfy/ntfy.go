package ntfy

import (
	"fmt"
	"net/http"
	"strings"
)

var NtfyChannel string

func SendNotification(message string) {
	if NtfyChannel != "" {
		_, err := http.Post(NtfyChannel, "text/plain", strings.NewReader(message))
		if err != nil {
			fmt.Printf("Error sending notification: %v\n", err)
		}
	}
}
