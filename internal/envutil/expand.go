package envutil

import (
	"os"
	"strings"
)

// ExpandWindowsEnv expands both Windows %VAR% and Unix $VAR / ${VAR}
// style environment variables in a string.
// os.ExpandEnv only handles $VAR and ${VAR} — this function additionally
// handles the %VAR% syntax that Windows users and configuration files use.
func ExpandWindowsEnv(s string) string {
	// First pass: expand %VARNAME% patterns.
	result := s
	for {
		start := strings.Index(result, "%")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], "%")
		if end == -1 {
			break
		}
		end += start + 1 // Adjust to absolute index.
		varName := result[start+1 : end]
		if varName == "" {
			// Escaped percent (%%) — collapse to single % and continue.
			result = result[:start] + "%" + result[end+1:]
			continue
		}
		value := os.Getenv(varName)
		result = result[:start] + value + result[end+1:]
	}

	// Second pass: expand $VAR and ${VAR} syntax.
	return os.ExpandEnv(result)
}
