package logs

import (
	"fmt"
	"sync/atomic"
)

var colorCodes = map[string]string{
	"Red":    "\033[31m",
	"Green":  "\033[32m",
	"Yellow": "\033[33m",
	"Reset":  "\033[0m",
}

var debugMode atomic.Bool

func SetDebug(enabled bool) {
	debugMode.Store(enabled)
}

func IsDebug() bool {
	return debugMode.Load()
}

func Error(format string, args ...any) {
	fmt.Printf("%s [%s] Error%s %s%s\n", colorCodes["Red"], callerInfo(), colorCodes["Reset"], fmt.Sprintf(format, args...), colorCodes["Reset"])
}

func Info(format string, args ...any) {
	fmt.Printf("%s [%s] Info%s %s%s\n", colorCodes["Green"], callerInfo(), colorCodes["Reset"], fmt.Sprintf(format, args...), colorCodes["Reset"])
}

func Debug(format string, args ...any) {
	if !debugMode.Load() {
		return
	}
	fmt.Printf("%s [%s] Debug%s %s%s\n", colorCodes["Yellow"], callerInfo(), colorCodes["Reset"], fmt.Sprintf(format, args...), colorCodes["Reset"])
}

func Success(format string, args ...any) {
	fmt.Printf("%s [%s] Success%s %s%s\n", colorCodes["Green"], callerInfo(), colorCodes["Reset"], fmt.Sprintf(format, args...), colorCodes["Reset"])
}

func Warning(format string, args ...any) {
	fmt.Printf("%s [%s] Warning%s %s%s\n", colorCodes["Yellow"], callerInfo(), colorCodes["Reset"], fmt.Sprintf(format, args...), colorCodes["Reset"])
}
