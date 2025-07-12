package burngormabs

import (
	"net/url"
	"os"
	"strings"

	"gorm.io/gorm"
)

type Model interface {
	GenericCRUDImpl[any]
}

type Init struct {
	Models   []any
	Db       *gorm.DB
	FilePath string
	FileName string
	Package  string // Definately gets overwritten depending on user input
}

var initialized *Init

// Define a generic implementation of CRUD operations
type GenericCRUDImpl[T any] struct {
	model *T // Hold a reference to the outer struct
}

func ValidateModel[T Model]() {}

// Define the GenericCRUD interface
type GenericCRUD[T Model] interface {
	Create() error
	Update() error
	Get() error
	GetAllSearch(url.Values) ([]T, error)
	GetSingleSearch(url.Values) error
	CountRecords(url.Values) (int64, error)
}

func GetRegisteredModels() []any {
	return initialized.Models
}

func (g GenericCRUDImpl[T]) Create() error {
	return initialized.Db.Create(g.model).Error
}

func (g GenericCRUDImpl[T]) Update(queries ...*gorm.DB) error {
	if len(queries) != 0 {
		return queries[0].Updates(g.model).Error
	}
	return initialized.Db.Updates(g.model).Error
}

func (g GenericCRUDImpl[T]) Get(queries ...*gorm.DB) error {
	if len(queries) != 0 {
		return queries[0].Where(g.model).First(g.model).Error
	}
	return initialized.Db.Where(g.model).First(g.model).Error
}

func (g GenericCRUDImpl[T]) GetAllSearch(searchFilter url.Values, queries ...*gorm.DB) ([]T, error) {
	var results []T
	if len(queries) != 0 {
		err := SearchMulti(searchFilter, queries[0], new(T), &results)
		if err != nil {
			return nil, err
		}
		return results, nil
	}
	err := SearchMulti(searchFilter, initialized.Db, new(T), &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (g GenericCRUDImpl[T]) GetSingleSearch(searchFilter url.Values) error {
	var result T
	err := SearchOne(searchFilter, initialized.Db, &result)
	if err != nil {
		return err
	}
	return nil
}

func (g GenericCRUDImpl[T]) CountRecords(searchFilter url.Values) (int64, error) {
	return Count(searchFilter, initialized.Db, new(T))
}

func Initialize(init *Init) {
	if len(init.Models) <= 0 {
		panic("no models declared")
	}
	if strings.EqualFold(initialized.FileName, "") {
		initialized.FileName = "genmodels.go"
	}

	if strings.EqualFold(init.Package, "") {
		initialized.Package = "models"
	}
	folders := strings.Split(init.FilePath, "/")
	if len(folders) <= 0 {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		dir += init.Package
		initialized.FilePath = dir
	}
	initialized = init
	// FIXME: Do the file

}
