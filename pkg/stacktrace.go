package loginjector

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

// StackTrace returns the current stack trace as a string.
func StackTrace() string {
	result := string(debug.Stack())
	result = strings.TrimSpace(result)
	return result
}

// ExtractMethodTrace extracts method trace information from a stack trace.
//
// This function processes the current stack trace and attempts to identify the
// location and method name based on the provided subpackages. It returns the
// method name along with its position and the remaining stack trace starting from
// that position.
//
// Parameters:
//   - subpackages: A variadic parameter that accepts one or more strings representing
//     subpackage paths. These are joined together to form the path pointer to be
//     searched within the stack trace.
//
// Returns:
//   - method and position (string): The method name followed by the file position (line number)
//     where it was found in the stack trace. Returns an empty string if no match is found.
//   - trace (string): The remaining stack trace starting from the identified position. Returns an empty
//     string if no match is found.
//
// Example usage:
//
//	method, trace := ExtractMethodTrace("stacktrace", "stacktrace.go")
//	fmt.Println("method: ", method)
//	fmt.Println("trace: ", trace)
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

const currentPackageFile = "loginjector/stacktrace.go"
