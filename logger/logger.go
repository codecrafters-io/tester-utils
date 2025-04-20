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

	// secondaryPrefix is a secondary prefix, that can be dynamically appended to the prefix.
	secondaryPrefix string

	logger log.Logger
}

// GetLogger Returns a logger.
func GetLogger(isDebug bool, prefix string) *Logger {
	color.NoColor = false

	coloredPrefix := yellowColorize(prefix)[0]
	return &Logger{
		logger:  *log.New(os.Stdout, coloredPrefix, 0),
		IsDebug: isDebug,
		prefix:  prefix,
	}
}

func (l *Logger) GetSecondaryPrefix() string {
	return l.secondaryPrefix
}

func (l *Logger) UpdateSecondaryPrefix(prefix string) {
	l.secondaryPrefix = prefix
	if prefix == "" {
		// Reset the prefix to the original one.
		l.logger.SetPrefix(yellowColorize(l.prefix)[0])
	} else {
		// Append the secondary prefix to the original one.
		l.logger.SetPrefix(yellowColorize(l.prefix + fmt.Sprintf("[%s] ", prefix))[0])
	}
}

func (l *Logger) ResetSecondaryPrefix() {
	l.UpdateSecondaryPrefix("")
}

// GetQuietLogger Returns a logger that only emits critical logs. Useful for anti-cheat stages.
func GetQuietLogger(prefix string) *Logger {
	color.NoColor = false

	coloredPrefix := yellowColorize(prefix)[0]
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
	for _, line := range successColorize(msg) {
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

	for _, line := range infoColorize(msg) {
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

	for _, line := range errorColorize(msg) {
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

	for _, line := range errorColorize(msg) {
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

	for _, line := range debugColorize(msg) {
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
