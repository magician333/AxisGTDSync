package api

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
)

func GetConfig() (configData ConfigType) {

	var config ConfigType
	config.PSQLURL = os.Getenv("psqlURL")

	if os.Getenv("corsURL") != "" {
		config.CorsURL = "https://www.axisgtd.work,http://localhost:3000/,http://127.0.0.1:8080," + os.Getenv("corsURL")
	} else {
		config.CorsURL = "https://www.axisgtd.work,http://localhost:3000/,http://127.0.0.1:8080"

	}

	if config.PSQLURL == "" {
		fmt.Println("Please set the environment variable psqlURL")
		fmt.Println("e.g. set psqlURL=\"user='youruser' password='yourpassword' dbname='yourdbname' sslmode='require'\"")
		os.Exit(0)
	}
	return config
}

func InitDB() {

	createUIDTableQuery := `
  	CREATE TABLE IF NOT EXISTS UID (
  		id serial NOT NULL,
  		name character varying(100) NOT NULL,
  		status BOOLEAN NOT NULL,
  		UNIQUE (name)
  	)`
	var err error
	_, err = db.Exec(createUIDTableQuery)
	checkerr(err)
}

func checkerr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func GenerateRandomHex(n int) (string, error) {
	bytes := make([]byte, (n+1)/2)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func GetName() (string, error) {
	uidName, err := GenerateRandomHex(5)
	checkerr(err)
	var exists bool

	query := "SELECT EXISTS(SELECT 1 FROM UID WHERE name = $1)"
	err = db.QueryRow(query, uidName).Scan(&exists)
	checkerr(err)

	for exists {
		uidName, err = GenerateRandomHex(5)
		checkerr(err)
		err = db.QueryRow(query, uidName).Scan(&exists)
		checkerr(err)
	}

	return uidName, nil
}

func DeleteDBRecord(uidName string, time int64) error {
	query := `
        DELETE FROM axisgtd
        WHERE uid_name = $1 AND time = $2;
    `

	result, err := db.Exec(query, uidName, time)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no records found with uid_name %s and time %d", uidName, time)
	}

	return nil
}

func DeleteUIDAndAxisGtdByUID(uidName string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	countQuery := `SELECT COUNT(*) FROM axisgtd WHERE uid_name = $1`
	var count int
	err = tx.QueryRow(countQuery, uidName).Scan(&count)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error checking count in axisgtd: %v", err)
	}

	if count > 0 {
		deleteAxisGtdQuery := `DELETE FROM axisgtd WHERE uid_name = $1`
		_, err = tx.Exec(deleteAxisGtdQuery, uidName)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error deleting from axisgtd: %v", err)
		}
	}

	deleteUIDQuery := `DELETE FROM uid WHERE name = $1`
	result, err := tx.Exec(deleteUIDQuery, uidName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error deleting from UID: %v", err)
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error getting affected rows from UID: %v", err)
	}
	if affectedRows == 0 {
		tx.Rollback()
		return fmt.Errorf("no UID record found for name %s", uidName)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}
