package burngormabs

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func SelectQueryBuilder(query *gorm.DB, parameters url.Values) (err error) {

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

func GormSearch(queryParams url.Values, query *gorm.DB) (q *gorm.DB, err error) {

	for name, param := range queryParams {
		value := strings.Split(name, "__")
		if len(value) != GORM_SEARCH_INPUT_COUNT {
			continue
		}
		columnOperation := value[0]
		columnName := value[1]

		columns := strings.Split(columnName, ":")
		paramVal := param[0]

		switch columnOperation {
		case "ilike":
			query.Where(compoundOr(columns, "%"+paramVal+"%", "ILIKE"))
		case "in":
			query.Where(compoundOrIn(columns, strings.Split(paramVal, ","), true))
		case "nin":
			query.Where(compoundOrIn(columns, strings.Split(paramVal, ","), false))
		case "gte":
			query.Where(compoundOr(columns, paramVal, ">="))
		case "lte":
			query.Where(compoundOr(columns, paramVal, "<="))
		case "gt":
			query.Where(compoundOr(columns, paramVal, ">"))
		case "lt":
			query.Where(compoundOr(columns, paramVal, "<"))
		case "eq":
			query.Where(compoundOr(columns, paramVal, "="))
		case "like":
			query.Where(compoundOr(columns, "%"+paramVal+"%", "LIKE"))
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

func compoundOr(columns []string, value interface{}, operator string) (clause string, args []interface{}) {
	var conditions []string
	for _, col := range columns {
		conditions = append(conditions, fmt.Sprintf("%s %s ?", col, operator))
		args = append(args, value)
	}
	clause = "(" + strings.Join(conditions, " OR ") + ")"
	return
}

func compoundOrIn(columns []string, values []string, include bool) (clause string, args []interface{}) {
	var conditions []string
	for _, col := range columns {
		if include {
			conditions = append(conditions, fmt.Sprintf("%s IN (?)", col))
		} else {
			conditions = append(conditions, fmt.Sprintf("%s NOT IN (?)", col))
		}
		args = append(args, values)
	}
	clause = "(" + strings.Join(conditions, " OR ") + ")"
	return
}
func SearchOne(parameters url.Values, database *gorm.DB, output any) (err error) {

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

func SearchMulti(parameters url.Values, database *gorm.DB, model any, output any) (err error) {
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

func Count(parameters url.Values, database *gorm.DB, model any) (count int64, err error) {
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
