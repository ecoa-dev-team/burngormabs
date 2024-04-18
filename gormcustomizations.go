package burngormabs

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func SelectQueryBuilder(query *gorm.DB, parameters map[string][]string) (err error) {

	query, err = GormSearch(parameters, query)
	if err != nil {
		return
	}
	if orderby, isorderd := parameters["orderby"]; isorderd {
		query = query.Order(orderby[0])
	}

	pagestr, ispage := parameters["page"]
	limitstr, islimit := parameters["size"]
	if !islimit || !ispage {
		limitstr = append(limitstr, "10")
		pagestr = append(pagestr, "1")
	}
	size, err := strconv.Atoi(limitstr[0])
	page, err1 := strconv.Atoi(pagestr[0])
	if err != nil || err1 != nil {
		page = 1
		size = 10
		err = nil
	}
	query.Offset(size * (page - 1)).Limit(size)
	return
}

func GormSearch(queryParams map[string][]string, query *gorm.DB) (q *gorm.DB, err error) {

	for name, param := range queryParams {
		value := strings.Split(name, "__")
		if len(value) != GORM_SEARCH_INPUT_COUNT {
			continue
		}
		columnOperation := value[0]
		columnName := value[1]

		switch columnOperation {
		case "ilike":
			query.Where(fmt.Sprintf("%s ILIKE ?", columnName), "%"+param[0]+"%")
		case "in":
			query.Where(fmt.Sprintf("%s IN (?)", columnName), strings.Split(param[0], ","))
		case "gte":
			query.Where(fmt.Sprintf("%s >= ?", columnName), param[0])
		case "lte":
			query.Where(fmt.Sprintf("%s <= ?", columnName), param[0])
		case "gt":
			query.Where(fmt.Sprintf("%s > ?", columnName), param[0])
		case "lt":
			query.Where(fmt.Sprintf("%s < ?", columnName), param[0])
		case "eq":
			query.Where(fmt.Sprintf("%s = ?", columnName), param[0])
		case "like":
			query.Where(fmt.Sprintf("%s LIKE ?", columnName), "%"+param[0]+"%")
		case "btwn":
			rangeBtwn := strings.Split(param[0], ",")
			if len(rangeBtwn) != RANGE_SEARCH_PARAM_COUNT {
				err = fmt.Errorf("range search requires 2 values received %v", param)
				return
			}

			query.Where(fmt.Sprintf("%s >= ? AND %s < ?", columnName, columnName), rangeBtwn[0], rangeBtwn[1])
		default:
			continue

		}
	}
	q = query
	return
}

func SearchOne(parameters map[string][]string, database *gorm.DB, output any) (err error) {

	query := database.Model(output)
	err = SelectQueryBuilder(query, parameters)

	if err != nil {
		return err
	}
	err = query.First(output).Error
	if err != nil {
		return err
	}
	return err
}

func SearchMulti(parameters map[string][]string, database *gorm.DB, model any, output any) (err error) {
	query := database.Model(model)
	err = SelectQueryBuilder(query, parameters)
	if err != nil {
		return
	}
	err = query.Find(output).Error
	if err != nil {
		return
	}
	return
}

func Count(parameters map[string][]string, database *gorm.DB, model any) (count int64, err error) {
	// Remove the pagination params
	query := database.Model(model)
	query, err = GormSearch(parameters, query)
	if err != nil {
		return
	}
	err = query.Count(&count).Error
	if err != nil {
		return
	}
	return
}
