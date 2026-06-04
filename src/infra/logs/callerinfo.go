package logs

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

var anonFuncRe = regexp.MustCompile(`\.\d+\.func\d+`)

func callerInfo() string {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	name := fn.Name()
	parts := strings.Split(name, "/")

	name = parts[len(parts)-1]

	name = anonFuncRe.ReplaceAllString(name, "")

	dotParts := strings.Split(name, ".")
	if len(dotParts) > 2 {
		name = strings.Join(dotParts[len(dotParts)-2:], ".")
	}

	fileParts := strings.Split(file, "/")
	fileName := fileParts[len(fileParts)-1]

	return fmt.Sprintf("%s %s:%d", name, fileName, line)
}
