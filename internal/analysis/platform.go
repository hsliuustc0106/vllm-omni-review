package analysis

import (
	"regexp"
	"strings"
)

// AnalyzePlatform scans for CUDA/device-specific patterns.
func AnalyzePlatform(diff string) []Hint {
	var hints []Hint
	lines := strings.Split(diff, "\n")

	hints = append(hints, checkCustomStream(lines)...)
	hints = append(hints, checkDevicePlacement(lines)...)
	hints = append(hints, checkLargeAllocation(lines)...)

	return hints
}

var streamCreateRE = regexp.MustCompile(`torch\.cuda\.Stream\s*\(`)

func checkCustomStream(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if streamCreateRE.MatchString(trimmed) {
			found := false
			end := i + 10
			if end > len(lines) {
				end = len(lines)
			}
			for j := i + 1; j < end; j++ {
				if strings.Contains(strings.TrimSpace(lines[j]), "synchronize()") {
					found = true
					break
				}
			}
			if !found {
				hints = append(hints, Hint{
					Severity: "medium",
					Pattern:  "custom-stream-no-sync",
					Line:     i + 1,
					Snippet:  trimmed,
					Message:  "Custom CUDA stream created without nearby synchronize()",
				})
			}
		}
	}
	return hints
}

var devicePlaceRE = regexp.MustCompile(`\.(cuda|to)\s*\(\s*["']cuda`)

func checkDevicePlacement(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if devicePlaceRE.MatchString(trimmed) {
			hints = append(hints, Hint{
				Severity: "low",
				Pattern:  "device-placement",
				Line:     i + 1,
				Snippet:  trimmed,
				Message:  "CUDA device placement: verify memory lifecycle and non-CUDA fallback",
			})
		}
	}
	return hints
}

var largeAllocRE = regexp.MustCompile(`torch\.(empty|zeros|ones|randn|full)\s*\(`)

func checkLargeAllocation(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if largeAllocRE.MatchString(trimmed) {
			hints = append(hints, Hint{
				Severity: "low",
				Pattern:  "large-allocation",
				Line:     i + 1,
				Snippet:  trimmed,
				Message:  "Large tensor allocation: verify memory budget and cleanup",
			})
		}
	}
	return hints
}

// AnalyzeAll runs all analyzers on a Python diff and returns combined formatted hints.
func AnalyzeAll(diff string) string {
	var all []Hint
	all = append(all, AnalyzePython(diff)...)
	all = append(all, AnalyzeTensorShapes(diff)...)
	all = append(all, AnalyzePlatform(diff)...)
	return FormatHints(all)
}
