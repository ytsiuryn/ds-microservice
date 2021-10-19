// Модуль сбора общих сведений о микросервисе:
//
// - даты последней модификации
//
// - списка используемых модулей и их версий

package microservice

import (
	"fmt"
	"os"
	"runtime/debug"

	collection "github.com/ytsiuryn/go-collection"
)

// BuildTime формирует строку даты создания/последней модификации исполняемого файла сервиса
// в указанном формате `fmt`.
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

// Modules возвращает список строк вида <module_path>@<version>.
// Список может быть отфильтрован конкретной выборкой модулей в `modNames`.
func Modules(modNames ...string) (lst []string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, dep := range bi.Deps {
		if len(modNames) == 0 || collection.ContainsStr(dep.Path, modNames) {
			lst = append(lst, fmt.Sprintf("%s@%s", dep.Path, dep.Version))
		}
	}
	return
}
