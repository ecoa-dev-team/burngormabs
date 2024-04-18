package burngormabs

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
)

var database *gorm.DB
var redis_c *redis.Client

type User struct {
	ID        uint      `json:"id" gorm:"column:id" binding:"required"`
	Username  string    `json:"username" gorm:"column:username" binding:"required"`
	FirstName string    `json:"firstname" gorm:"column:firstname" binding:"required"`
	LastName  string    `json:"lastname" gorm:"column:lastname" binding:"required"`
	CreatedAt time.Time `json:"createdat" gorm:"column:createdat" binding:"required"`
}

func (u User) GetTable() string {
	return "users"
}

func setup() {
	mr, err := miniredis.Run()
	if err != nil {
		return
	}
	redis_c = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	// NOTE: Database setup
	db, err := gorm.Open(sqlite.Open("file:memdb1?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return
	}
	database = db
	database.AutoMigrate(User{})
	// NOTE: Data injestion
	data := []User{
		{
			ID:        1,
			Username:  "test1",
			FirstName: "test",
			LastName:  "one",
			CreatedAt: time.Date(2024, time.March, 1, 7, 0, 0, 0, time.Local),
		},
		{
			ID:        2,
			Username:  "test2",
			FirstName: "test",
			LastName:  "two",
			CreatedAt: time.Date(2024, time.March, 1, 8, 0, 0, 0, time.Local),
		},
		{
			ID:        3,
			Username:  "test3",
			FirstName: "test",
			LastName:  "Three",
			CreatedAt: time.Date(2024, time.March, 1, 9, 0, 0, 0, time.Local),
		},
		{
			ID:        4,
			Username:  "test4",
			FirstName: "test",
			LastName:  "four",
			CreatedAt: time.Date(2024, time.March, 1, 10, 0, 0, 0, time.Local),
		},
	}
	database.Create(&data)
	fmt.Printf("\033[1;33m%s\033[0m", "> Setup completed")
	fmt.Printf("\n")
}

func teardown() {
	// Do something here.
	fmt.Printf("\033[1;33m%s\033[0m", "> Teardown completed")
	fmt.Printf("\n")
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestPersistance(t *testing.T) {
	users := []User{}
	err := database.Find(&users).Error
	assert.NoError(t, err)
	assert.Equal(t, 4, len(users), "they should be equal")
}
func TestSearchOne(t *testing.T) {
	user := User{}
	err := SearchOne(map[string][]string{"eq__lastname": {"two"}}, database, &user)
	assert.NoError(t, err)
	assert.Equal(t, "test", user.FirstName, "they should be equal")
}

func TestSearchMulti(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"like__lastname": {"t"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users), "they should be equal")
}

func TestSearchMultiIn(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"in__id": {"1,2"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users), "they should be equal")
}

func TestSearchMultiLt(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"lt__id": {"2"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users), "they should be equal")
}

func TestSearchMultiLte(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"lte__id": {"2"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users), "they should be equal")
}

func TestSearchMultiPagination(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"page": {"1"}, "size": {"1"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users), "they should be equal")
}
func TestSearchMultiPaginationError(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"page": {"a"}, "size": {"b"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(users), "they should be equal")
}
func TestSearchMultiOrderBy(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"lte__id": {"2"}, "orderby": {"id DESC"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, uint(2), users[0].ID, "they should be equal")
}

func TestSearchMultiGte(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"gte__id": {"2"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(users), "they should be equal")
}

func TestSearchMultiGt(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"gt__id": {"2"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users), "they should be equal")
}

func TestSearchMultiBtwn(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"btwn__createdat": {"2024-03-01 07:00:00,2024-03-01 09:00:00"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users), "they should be equal")
}

func TestSearchMultiBtwnErr(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"btwn__createdat": {"2024-03-01 09:00:00"}}, database, User{}, &users)
	assert.Error(t, err)
}
func TestSearchMultiInvalidParam(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"nggn__createdat": {"2024-03-01 09:00:00"}}, database, User{}, &users)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(users), "they should be equal")
}

func TestSearchOneCache(t *testing.T) {
	user := User{}
	err := SearchOne(map[string][]string{"eq__id": {"1"}}, database, &user)
	assert.NoError(t, err)
	err = SearchOne(map[string][]string{"eq__id": {"1"}}, database, &user)
	assert.NoError(t, err)
	assert.Equal(t, "test1", user.Username, "they should be equal")
}

func TestCount(t *testing.T) {
	count, err := Count(map[string][]string{"like__lastname": {"f"}}, database, User{})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "they should be equal")
}

func TestCountNoMatch(t *testing.T) {
	count, err := Count(map[string][]string{"eq__id": {"10"}}, database, User{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "they should be equal")
}

func TestCountPagination(t *testing.T) {
	count, err := Count(map[string][]string{"page": {"1"}, "size": {"2"}}, database, User{})
	assert.NoError(t, err)
	assert.Equal(t, int64(4), count, "they should be equal")
}

func TestSearchOneBtwnError(t *testing.T) {
	user := User{}
	err := SearchOne(map[string][]string{"btwn__createdat": {"2024-03-01 09:00:00"}}, database, &user)
	assert.Error(t, err)

}

func TestSearchOneMissingCol(t *testing.T) {
	user := User{}
	err := SearchOne(map[string][]string{"eq__invalidcol": {"2024-03-01 09:00:00"}}, database, &user)
	assert.Error(t, err)

}

func TestSearchMultiInvalidCol(t *testing.T) {
	users := []User{}
	err := SearchMulti(map[string][]string{"eq__invalidcol": {"2024-03-01 09:00:00"}}, database, User{}, &users)
	assert.Error(t, err)
}

func TestCountInvalidCol(t *testing.T) {
	_, err := Count(map[string][]string{"eq__invalidcol": {"2024-03-01 09:00:00"}}, database, User{})
	assert.Error(t, err)
}

func TestCountBtwnError(t *testing.T) {
	_, err := Count(map[string][]string{"btwn__createdat": {"2024-03-01 09:00:00"}}, database, User{})
	assert.Error(t, err)

}
