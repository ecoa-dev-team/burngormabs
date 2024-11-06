package burngormabs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CacheOptions struct {
	CheckCache bool
	TTL        time.Duration
	Client     *redis.Client
}

type IModel interface {
	GetTable() string
}

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
			query = addOrConditions(query, columnName, param, false)
		case "eqnot":
			query = addOrConditions(query, columnName, param, true)
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

// Helper function to handle eq and eqnot conditions with OR
func addOrConditions(query *gorm.DB, columnName string, param []string, negate bool) *gorm.DB {
	var orConditions []string
	var values []interface{}

	params := strings.Split(param[0], ",")
	for _, p := range params {
		p = strings.TrimSpace(p)

		if strings.ToLower(p) == "null" {
			if negate {
				orConditions = append(orConditions, fmt.Sprintf("%s IS NOT NULL", columnName))
			} else {
				orConditions = append(orConditions, fmt.Sprintf("%s IS NULL", columnName))
			}
		} else {
			if negate {
				orConditions = append(orConditions, fmt.Sprintf("%s != ?", columnName))
			} else {
				orConditions = append(orConditions, fmt.Sprintf("%s = ?", columnName))
			}
			values = append(values, p)
		}
	}

	if len(orConditions) > 0 {
		query = query.Where(strings.Join(orConditions, " OR "), values...)
	}

	return query
}

func SearchOne(parameters map[string][]string, database *gorm.DB, model IModel, output any, opts CacheOptions) (err error) {
	key := model.GetTable()
	for key := range parameters {
		key += fmt.Sprintf("|%s,%s", key, parameters[key][0])
	}
	if opts.CheckCache {
		res, err := opts.Client.Get(context.Background(), key).Result()
		err2 := json.Unmarshal([]byte(res), output)
		if err == nil && err2 == nil {
			return err
		}
	}
	query := database.Table(model.GetTable())
	err = SelectQueryBuilder(query, parameters)

	if err != nil {
		return err
	}
	err = query.First(output).Error
	if err != nil {
		return err
	}
	if opts.CheckCache {
		trxb, err := json.Marshal(output)
		if err != nil {
			return err
		}
		err = opts.Client.Set(context.Background(), key, trxb, time.Hour).Err()
		if err != nil {
			return nil
		}
	}
	return err
}

func SearchMulti(parameters map[string][]string, database *gorm.DB, model IModel, output any) (err error) {
	query := database.Table(model.GetTable())
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

func Count(parameters map[string][]string, database *gorm.DB, model IModel) (count int64, err error) {
	// Remove the pagination params
	query := database.Table(model.GetTable())
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
