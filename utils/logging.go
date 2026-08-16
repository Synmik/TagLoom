package utils

import (
	"fmt"
	"os"
)

// Tiny shared logger so all backend console output has a consistent shape:
//
//	[TagLoom] LEVEL: message
//
// Levels:
//   - LogInfo — normal noteworthy events (migrations, cleanup summaries)
//   - LogWarn — non-fatal problems worth investigating (best-effort failures)
//
// Output goes to stderr, matching the previous fmt.Printf behaviour.

const logPrefix = "[TagLoom]"

func logf(level, format string, args ...any) {
	fmt.Fprintf(os.Stderr, logPrefix+" %s: "+format+"\n", append([]any{level}, args...)...)
}

// LogInfo logs a normal noteworthy event.
func LogInfo(format string, args ...any) {
	logf("INFO", format, args...)
}

// LogWarn logs a non-fatal problem.
func LogWarn(format string, args ...any) {
	logf("WARN", format, args...)
}
