package logger

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

func colorize(colorToUse color.Attribute, fstring string, args ...any) []string {
	var msg string

	if len(args) == 0 {
		msg = fstring // Treat as plain string if no args
	} else {
		msg = fmt.Sprintf(fstring, args...) // Format if args are present
	}

	lines := strings.Split(msg, "\n")
	colorizedLines := make([]string, len(lines))

	for i, line := range lines {
		colorizedLines[i] = color.New(colorToUse).SprintFunc()(line)
	}

	return colorizedLines
}

func debugColorize(fstring string, args ...any) []string {
	return colorize(color.FgCyan, fstring, args...)
}

func infoColorize(fstring string, args ...any) []string {
	return colorize(color.FgHiBlue, fstring, args...)
}

func successColorize(fstring string, args ...any) []string {
	return colorize(color.FgHiGreen, fstring, args...)
}

func errorColorize(fstring string, args ...any) []string {
	return colorize(color.FgHiRed, fstring, args...)
}

func yellowColorize(fstring string, args ...any) []string {
	return colorize(color.FgYellow, fstring, args...)
}

// Logger is a wrapper around log.Logger with the following features:
//   - Supports a prefix
//   - Adds colors to the output
//   - Debug mode (all logs, debug and above)
//   - Quiet mode (only critical logs)
type Logger struct {
	// IsDebug is used to determine whether to emit debug logs.
	IsDebug bool

	// IsQuiet is used to determine whether to emit non-critical logs.
	IsQuiet bool

	// prefix is the prefix to be used for all logs.
	prefix string

	// secondaryPrefixes is a slice of prefixes that are printed after Logger.prefix
	secondaryPrefixes []string

	logger log.Logger
}

// GetLogger Returns a logger.
func GetLogger(isDebug bool, prefix string) *Logger {
	color.NoColor = false

	coloredPrefix := yellowColorize("%s", prefix)[0]
	return &Logger{
		logger:  *log.New(os.Stdout, coloredPrefix, 0),
		IsDebug: isDebug,
		prefix:  prefix,
	}
}

// GetSecondaryPrefix returns the first secondary prefix
func (l *Logger) GetSecondaryPrefix() string {
	if len(l.secondaryPrefixes) == 0 {
		return ""
	}
	return l.secondaryPrefixes[0]
}

// UpdateSecondaryPrefix replaces all secondary prefixes with the new one
// for backward compatibility
func (l *Logger) UpdateSecondaryPrefix(prefix string) {
	l.secondaryPrefixes = []string{}
	if prefix != "" {
		l.secondaryPrefixes = append(l.secondaryPrefixes, prefix)
	}
	l.updateLoggerPrefix()
}

// ResetSecondaryPrefix clears all secondary prefixes
func (l *Logger) ResetSecondaryPrefix() {
	// Backward compatibility: clear all secondary prefixes
	l.secondaryPrefixes = []string{}
	l.updateLoggerPrefix()
}

// updateLoggerPrefix updates the logger's prefix based on all secondary prefixes
func (l *Logger) updateLoggerPrefix() {
	if len(l.secondaryPrefixes) == 0 {
		l.logger.SetPrefix(yellowColorize("%s", l.prefix)[0])
	} else {
		fullPrefix := l.prefix
		for _, secondaryPrefix := range l.secondaryPrefixes {
			fullPrefix += fmt.Sprintf("[%s] ", secondaryPrefix)
		}
		l.logger.SetPrefix(yellowColorize("%s", fullPrefix)[0])
	}
}

// PushSecondaryPrefix adds a new secondary prefix to the end of secondaryPrefixes slice
func (l *Logger) PushSecondaryPrefix(prefix string) {
	l.secondaryPrefixes = append(l.secondaryPrefixes, prefix)
	l.updateLoggerPrefix()
}

// PopSecondaryPrefix removes the last secondary prefix from the secondaryPrefixes slice
func (l *Logger) PopSecondaryPrefix() string {
	if len(l.secondaryPrefixes) == 0 {
		return ""
	}
	lastPrefix := l.secondaryPrefixes[len(l.secondaryPrefixes)-1]
	l.secondaryPrefixes = l.secondaryPrefixes[:len(l.secondaryPrefixes)-1]
	l.updateLoggerPrefix()
	return lastPrefix
}

// WithAddtionalSecondaryPrefix is helpful you want to run
// one or more logging statements using an additional secondary prefix
func (l *Logger) WithAdditionalSecondaryPrefix(prefix string, fn func()) {
	l.PushSecondaryPrefix(prefix)
	defer l.PopSecondaryPrefix()
	fn()
}

// GetQuietLogger Returns a logger that only emits critical logs. Useful for anti-cheat stages.
func GetQuietLogger(prefix string) *Logger {
	color.NoColor = false

	coloredPrefix := yellowColorize("%s", prefix)[0]
	return &Logger{
		logger:  *log.New(os.Stdout, coloredPrefix, 0),
		IsDebug: false,
		IsQuiet: true,
		prefix:  prefix,
	}
}

func (l *Logger) Successf(fstring string, args ...any) {
	if l.IsQuiet {
		return
	}

	for _, line := range successColorize(fstring, args...) {
		l.logger.Println(line)
	}
}

func (l *Logger) Successln(msg string) {
	if l.IsQuiet {
		return
	}
	for _, line := range successColorize("%s", msg) {
		l.logger.Println(line)
	}
}

func (l *Logger) Infof(fstring string, args ...any) {
	if l.IsQuiet {
		return
	}

	for _, line := range infoColorize(fstring, args...) {
		l.logger.Println(line)
	}
}

func (l *Logger) Infoln(msg string) {
	if l.IsQuiet {
		return
	}

	for _, line := range infoColorize("%s", msg) {
		l.logger.Println(line)
	}
}

// Criticalf is to be used only in anti-cheat stages
func (l *Logger) Criticalf(fstring string, args ...any) {
	if !l.IsQuiet {
		panic("Critical is only for quiet loggers")
	}

	for _, line := range errorColorize(fstring, args...) {
		l.logger.Println(line)
	}
}

// Criticalln is to be used only in anti-cheat stages
func (l *Logger) Criticalln(msg string) {
	if !l.IsQuiet {
		panic("Critical is only for quiet loggers")
	}

	for _, line := range errorColorize("%s", msg) {
		l.logger.Println(line)
	}
}

func (l *Logger) Errorf(fstring string, args ...any) {
	if l.IsQuiet {
		return
	}

	for _, line := range errorColorize(fstring, args...) {
		l.logger.Println(line)
	}
}

func (l *Logger) Errorln(msg string) {
	if l.IsQuiet {
		return
	}

	for _, line := range errorColorize("%s", msg) {
		l.logger.Println(line)
	}
}

func (l *Logger) Debugf(fstring string, args ...any) {
	if !l.IsDebug {
		return
	}

	for _, line := range debugColorize(fstring, args...) {
		l.logger.Println(line)
	}
}

func (l *Logger) Debugln(msg string) {
	if !l.IsDebug {
		return
	}

	for _, line := range debugColorize("%s", msg) {
		l.logger.Println(line)
	}
}

func (l *Logger) Plainf(fstring string, args ...any) {
	formattedString := fmt.Sprintf(fstring, args...)

	for line := range strings.SplitSeq(formattedString, "\n") {
		l.logger.Println(line)
	}
}

func (l *Logger) Plainln(msg string) {
	lines := strings.SplitSeq(msg, "\n")

	for line := range lines {
		l.logger.Println(line)
	}
}
