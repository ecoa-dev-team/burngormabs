package utilities

import (
	"path"
	"reflect"
	"runtime"
)

type StructInfo struct {
	Name       string
	Package    string
	FolderPath string
}

func GetStructInfo(i any) StructInfo {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	pc := reflect.ValueOf(i).Pointer()
	fn := runtime.FuncForPC(pc)

	var folder string
	if fn != nil {
		_, file, _, ok := runtime.Caller(1)
		if ok {
			folder = path.Dir(file)
		}
	}

	return StructInfo{
		Name:       t.Name(),
		Package:    t.PkgPath(),
		FolderPath: folder,
	}
}
