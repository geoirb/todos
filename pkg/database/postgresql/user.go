package postgresql

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/geoirb/todos/pkg/database"
	"github.com/geoirb/todos/pkg/storage"
)

// User database.
type User struct {
	mutex sync.Mutex
	db    *sqlx.DB

	insertUser     string
	selectUser     string
	selectUserList string

	connect func() (*sqlx.DB, error)
}

var _ database.User = &User{}

func NewUser(
	dbDriver string,
	connectLayout string,
	host string,
	port int,
	database string,
	user string,
	password string,

	insertUser string,
	selectUser string,
	selectUserList string,
) (u *User, err error) {
	u = &User{
		insertUser:     insertUser,
		selectUser:     selectUser,
		selectUserList: selectUserList,
	}
	connectCfg := fmt.Sprintf(connectLayout, host, port, user, password, database)
	u.connect = func() (*sqlx.DB, error) {
		return sqlx.Connect(dbDriver, connectCfg)
	}
	u.db, err = u.connect()
	return
}

func (u *User) Insert(ctx context.Context, user storage.UserInfo) (err error) {
	if err = u.check(); err != nil {
		return
	}
	_, err = u.db.QueryContext(ctx, u.insertUser, user.Email, user.Password, true)
	return
}

func (u *User) SelectOne(ctx context.Context, filter storage.UserFilter) (user storage.UserInfo, err error) {
	if err = u.check(); err != nil {
		return
	}

	if filter.Email == nil || filter.Password == nil {
		err = fmt.Errorf("not found params")
		return
	}

	var dbUser UserInfo
	err = u.db.GetContext(ctx, &dbUser, u.selectUser, *filter.Email, *filter.Password)
	user = storage.UserInfo(dbUser)

	return
}

func (u *User) SelectList(ctx context.Context, filter storage.UserFilter) (users []storage.UserInfo, err error) {
	if err = u.check(); err != nil {
		return
	}

	idArg, emailArg := "*", "*"
	if filter.ID != nil {
		idArg = *filter.ID
	}

	if filter.Email != nil {
		emailArg = *filter.Email
	}

	var dbUsers []UserInfo
	err = u.db.SelectContext(ctx, &dbUsers, u.selectUserList, idArg, emailArg)

	users = make([]storage.UserInfo, 0, len(dbUsers))
	for _, user := range dbUsers {
		users = append(users, storage.UserInfo(user))
	}
	return
}

func (u *User) Close() error {
	return u.db.Close()
}

func (u *User) check() (err error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if err = u.db.Ping(); err != nil {
		if u.db, err = u.connect(); err != nil {
			err = fmt.Errorf("connect db %s", err)
		}
	}
	return
}
