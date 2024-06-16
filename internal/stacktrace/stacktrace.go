package stacktrace

import (
	"path"
	"regexp"
	"runtime/debug"
	"strings"
)

var (
	rxFileRowNumber = regexp.MustCompile(`(\w+/\w+/\w+.go:\d+)\s+[A-Ba-z0-9+.]+$`)
	rxMethodName    = regexp.MustCompile(`/(\w+[^/]+\w+)[({]+.+$`)
)

func StackTrace() string {
	result := string(debug.Stack())
	result = strings.TrimSpace(result)
	return result
}

func ExtractMethodTrace(subpackages ...string) (string, string) {
	pointer := currentPackageFile
	if subpackages != nil && len(subpackages) > 0 {
		pointer = path.Join(subpackages...)
	}

	stacktrace := StackTrace()

	// extract the first line of the stacktrace
	if index := strings.LastIndex(stacktrace, pointer); index > 0 {
		index += strings.Index(stacktrace[index:], "\n") + 1
		lines := strings.Split(stacktrace[index:], "\n")
		method := ""
		position := ""
		trace := make([]string, 0, len(lines))
		for l, line := range lines {
			parts := rxFileRowNumber.FindAllStringSubmatch(line, -1)
			if len(parts) == 1 && len(parts[0]) >= 2 {
				position = strings.TrimSpace(parts[0][1])
				//trace = strings.Join(lines[l:], "\n")
				for i, iCount := l, len(lines); i < iCount; i++ {
					trace = append(trace, strings.TrimSpace(lines[i]))
				}
				break
			}
			parts = rxMethodName.FindAllStringSubmatch(line, -1)
			if len(parts) == 1 && len(parts[0]) >= 2 {
				method = strings.TrimSpace(parts[0][1]) + "\n"
			}
		}
		return method + position, strings.Join(trace, "\n")
	}

	return "", ""
}

const currentPackageFile = "stacktrace/stacktrace.go"
