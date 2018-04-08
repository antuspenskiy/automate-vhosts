package db

import (
	"database/sql"
	"fmt"
)

// DropSalary clear data before using database
func DropSalary(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("UPDATE %s.user_data SET salary = 10000, salary_proposed = 11000;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DropDB drop MySQL database
func DropDB(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DropUser drop user
func DropUser(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("DROP USER '%s'@'localhost';", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CreateDB create MySQL database
func CreateDB(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GrantUserPriv grant user privileges to MySQL DB
func GrantUserPriv(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';", dbname, dbname, dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// FlushPriv flush the privileges
func FlushPriv(db *sql.DB) (int64, error) {
	res, err := db.Exec("FLUSH PRIVILEGES;")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
