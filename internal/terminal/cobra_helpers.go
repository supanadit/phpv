package terminal

import "strings"

func parseExtensions(extStr string) []string {
	if extStr == "" {
		return nil
	}
	extensions := []string{}
	for _, ext := range strings.Split(extStr, ",") {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			extensions = append(extensions, ext)
		}
	}
	return extensions
}
