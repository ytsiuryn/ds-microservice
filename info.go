package microservice

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	collection "github.com/ytsiuryn/go-collection"
)

func BuildTime(fmt string) string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fi.ModTime().Format(fmt)
}

func Modules(modNames []string) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	lst := []string{}
	for _, dep := range bi.Deps {
		if len(modNames) == 0 || collection.ContainsStr(dep.Path, modNames) {
			lst = append(lst, fmt.Sprintf("%s/%s", dep.Path, dep.Version))
		}
	}
	return strings.Join(lst, ", ")
}
