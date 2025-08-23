package db

import (
	"time"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type UserState int
const (
	UserStatePending = iota
	UserStateLoggedIn
)

type User struct {
	Id          string    `json:"phone"`
	State       UserState `json:"state"`
	RegDate     time.Time `json:"reg-date"`
	Otp         Otp       `json:"otp"`
}

type Otp struct {
	Val       string     `json:"val"`
	ExpiresAt time.Time  `json:"expires-at"`
	Tries     int        `json:"tries"`
	FirstTry  time.Time  `json:"first-try"`
}

type DataBase struct {
	db *sql.DB
}

func NewDb() (*DataBase, error) {
	// root:db@tcp(localhost:3306)/db
	db, err := sql.Open("mysql", "root:db@unix(/var/run/mysqld/mysqld.sock)/db?parseTime=true")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Zero initialized time.Time is valid and we want to store it.
	if _, err := db.Exec(`SET SESSION sql_mode = REPLACE(
		REPLACE(@@SESSION.sql_mode, 'NO_ZERO_IN_DATE', ''), 'NO_ZERO_DATE', '')`,
	); err != nil {
		return nil, err
	}

	return &DataBase{db}, nil
}

func (d *DataBase) GetUser(id string) (*User, error) {
    row := d.db.QueryRow(`
        SELECT id, state, reg_date, otp_val, otp_exp, otp_tries, otp_first
        FROM Users WHERE id = ?`, id)

    u := &User{}
    err := row.Scan(&u.Id, &u.State, &u.RegDate, &u.Otp.Val, &u.Otp.ExpiresAt, &u.Otp.Tries, &u.Otp.FirstTry)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

    return u, nil
}

func (d *DataBase) SaveUser(u *User) error {
    _, err := d.db.Exec(`
        INSERT INTO Users (id, state, reg_date, otp_val, otp_exp, otp_tries, otp_first)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            state=VALUES(state),
            reg_date=VALUES(reg_date),
            otp_val=VALUES(otp_val),
            otp_exp=VALUES(otp_exp),
            otp_tries=VALUES(otp_tries),
            otp_first=VALUES(otp_first)
    `, u.Id, u.State, u.RegDate, u.Otp.Val, u.Otp.ExpiresAt, u.Otp.Tries, u.Otp.FirstTry)
    return err
}

func (d *DataBase) ListUsers(offset, limit int) ([]*User, error) {
    rows, err := d.db.Query(`
        SELECT id, state, reg_date, otp_val, otp_exp, otp_tries, otp_first
        FROM Users
        ORDER BY id ASC
        LIMIT ? OFFSET ?`, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    users := []*User{}
    for rows.Next() {
        u := &User{}
		err := rows.Scan(&u.Id, &u.State, &u.RegDate, &u.Otp.Val, &u.Otp.ExpiresAt, &u.Otp.Tries, &u.Otp.FirstTry);
		if err != nil {
			return nil, err
		}
        users = append(users, u)
    }
    return users, nil
}
