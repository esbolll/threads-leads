package config

import (
	"os"
	"strings"
)

// WriteEnvValue sets KEY=value in a dotenv file, keeping other lines intact.
func WriteEnvValue(path, key, value string) error {
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+1)
	replaced := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			out = append(out, key+"="+value)
			replaced = true
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, key+"="+value)
	}
	content := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}
