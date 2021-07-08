package microservice

import (
	"errors"
	"os"
	"runtime/debug"
	"time"

	collection "github.com/ytsiuryn/go-collection"
)

// ServiceInfo хранит общие сведения о микросервисе.
type ServiceInfo struct {
	Subsystem, Name, Description string
	Built                        time.Time
	Dependencies                 map[string]string
}

func (si ServiceInfo) Init(info ServiceInfo) error {
	var err error
	si = info
	if err = si.setBuildTime(); err != nil {
		return err
	}
	if si.Dependencies, err = modules(); err != nil {
		return err
	}
	return nil
}

func (si ServiceInfo) setBuildTime() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	si.Built = fi.ModTime()
	return nil
}

// BuildTime возвращает форматированную строку даты и времени сборки.
func (si ServiceInfo) BuildTime(fmt string) string {
	return si.Built.Format(fmt)
}

// ModDeps возвращает отфильтрованный список используемых в микросервисе модулей с информацией
// об их версии.
// Если параметр `modNames` пустой или равен nil, возвращается описание по всем модулям.
func (si ServiceInfo) ModDeps(modNames []string) map[string]string {
	if modNames == nil {
		modNames = []string{}
	}
	ret := map[string]string{}
	for path, ver := range si.Dependencies {
		if len(modNames) == 0 || collection.ContainsStr(path, modNames) {
			ret[path] = ver
		}
	}
	return ret
}

// Возвращает map, где ключом является наименование модуля, а значением - его версия.
func modules() (map[string]string, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("Failed to read build info")
	}
	modInfos := map[string]string{}
	for _, dep := range bi.Deps {
		modInfos[dep.Path] = dep.Version
	}
	return modInfos, nil
}
