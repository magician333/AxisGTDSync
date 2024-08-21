package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	_ "github.com/lib/pq"
)

type AxisGTDType struct {
	Todolist string `json:"todolist"`
	Config   string `json:"config"`
	Time     int64  `json:"time"`
	UIDName  string `json:"uidname"`
}
type UID struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

type AxisGTDJsonType struct {
	Name     string `json:"name"`
	Status   bool   `json:"status"`
	Todolist string `json:"todolist"`
	Config   string `json:"config"`
	Time     int64  `json:"time"`
}

type IDStatus struct {
	Status       bool `json:"status"`
	AxisgtdCount int  `json:"axisgtdCount"`
}

type IDJsonType struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
	Count  int    `json:"count"`
}

type IDSType struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
	Count  int    `json:"count"`
}

func GetConfig() (psqlUrl string, corsUrl string) {
	configPath := "./config.json"

	content, err := ioutil.ReadFile(configPath)
	checkerr(err)

	var config map[string]string
	err = json.Unmarshal(content, &config)
	checkerr(err)

	return config["psqlUrl"], config["corsUrl"]
}

func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, (n+1)/2)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func getName(db *sql.DB) (string, error) {
	uidName, err := generateRandomHex(5)
	checkerr(err)
	var exists bool

	query := "SELECT EXISTS(SELECT 1 FROM UID WHERE name = $1)"
	err = db.QueryRow(query, uidName).Scan(&exists)
	checkerr(err)

	for exists {
		uidName, err = generateRandomHex(5)
		checkerr(err)
		err = db.QueryRow(query, uidName).Scan(&exists)
		checkerr(err)
	}

	return uidName, nil
}

func deleteRecord(db *sql.DB, uidName string, time int64) error {
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

func deleteUIDAndAxisGtdByUID(db *sql.DB, uidName string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	deleteAxisGtdQuery := `
        DELETE FROM axisgtd
        WHERE uid_name = $1;
    `
	result, err := tx.Exec(deleteAxisGtdQuery, uidName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error deleting from axisgtd: %v", err)
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error getting affected rows from axisgtd: %v", err)
	}
	if affectedRows == 0 {
		tx.Rollback()
		return fmt.Errorf("no axisgtd records found for uid_name %s", uidName)
	}

	deleteUIDQuery := `
        DELETE FROM UID
        WHERE name = $1;
    `
	result, err = tx.Exec(deleteUIDQuery, uidName)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error deleting from UID: %v", err)
	}
	affectedRows, err = result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error getting affected rows from UID: %v", err)
	}
	if affectedRows == 0 {
		tx.Rollback()
		return fmt.Errorf("no UID record found for name %s", uidName)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

func main() {
	psqlUrl, corsUrl := GetConfig()

	db, err := sql.Open("postgres", psqlUrl)
	checkerr(err)
	createUIDTableQuery := `
  	CREATE TABLE IF NOT EXISTS UID (
  		id serial NOT NULL,
  		name character varying(100) NOT NULL,
  		status BOOLEAN NOT NULL,
  		UNIQUE (name)
  	)`
	_, err = db.Exec(createUIDTableQuery)
	checkerr(err)

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: corsUrl,
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("AxisGTD synchronization service has been run successfully!")
	})

	app.Get("/create", func(c *fiber.Ctx) error {

		uidName, err := getName(db)
		checkerr(err)

		query := `INSERT INTO UID (name,status) VALUES ($1,$2)`
		_, err = db.Exec(query, uidName, true)
		checkerr(err)

		createDataTableQuery := `
    	CREATE TABLE IF NOT EXISTS axisgtd (
        	todolist TEXT NOT NULL,
        	config TEXT NOT NULL,
        	time BIGINT NOT NULL,
        	uid_name character varying(100) NOT NULL,
        	CONSTRAINT fk_uid_name FOREIGN KEY (uid_name) REFERENCES UID(name) 
    	);`

		_, err = db.Exec(createDataTableQuery)
		checkerr(err)
		msg := fmt.Sprintf("Create ID successful! Your ID is %s", uidName)

		return c.SendString(msg)
	})

	app.Get("/id/:name", func(c *fiber.Ctx) error {

		query := `
		SELECT
			axisgtd.*, 
    		UID.status
		FROM
			axisgtd
		JOIN 
    		UID ON axisgtd.uid_name = UID.name
		WHERE
			uid_name = $1	
		`

		rows, err := db.Query(query, c.Params("name"))
		checkerr(err)
		defer rows.Close()

		var dataList []AxisGTDJsonType

		for rows.Next() {
			var axisgtd AxisGTDType
			var uid UID
			err := rows.Scan(&axisgtd.Todolist,
				&axisgtd.Config,
				&axisgtd.Time,
				&uid.Name,
				&uid.Status)
			checkerr(err)
			if uid.Status {
				dataList = append(dataList, AxisGTDJsonType{
					Todolist: axisgtd.Todolist,
					Config:   axisgtd.Config,
					Time:     axisgtd.Time,
				})
				if len(dataList) == 0 {
					return c.SendStatus(404)
				}

			} else {
				return c.SendStatus(404)
			}

		}

		jsonData, err := json.Marshal(dataList)
		checkerr(err)
		return c.JSON(string(jsonData))
	})

	app.Get("/ids", func(c *fiber.Ctx) error {

		query := `
        SELECT
			UID.name,
			UID.status,
    		COUNT(*) AS axisgtd_count
		FROM
    		UID
		LEFT JOIN axisgtd ON UID.name = axisgtd.uid_name
		GROUP BY
			UID.name,
    		UID.status;`

		rows, err := db.Query(query)
		checkerr(err)
		var ids []IDSType
		for rows.Next() {
			var preID IDSType

			rows.Scan(&preID.Name, &preID.Status, &preID.Count)
			ids = append(ids, IDSType{
				Name:   preID.Name,
				Status: preID.Status,
				Count:  preID.Count,
			})
		}
		if len(ids) == 0 {
			return c.SendStatus(404)
		}
		jsonData, err := json.Marshal(ids)
		checkerr(err)
		return c.JSON(string(jsonData))
	})

	app.Get("/sync/:name", func(c *fiber.Ctx) error {
		query := `SELECT 
            axisgtd.*, 
            UID.name,
			UID.status
        FROM 
            axisgtd 
        JOIN 
            UID 
        ON 
            axisgtd.uid_name = UID.name
		WHERE
			UID.name =$1
		ORDER BY
			time DESC
		LIMIT 1;`
		rows, err := db.Query(query, c.Params("name"))

		checkerr(err)
		defer rows.Close()
		for rows.Next() {
			var axisgtd AxisGTDType
			var uid UID
			err := rows.Scan(&axisgtd.Todolist,
				&axisgtd.Config,
				&axisgtd.Time,
				&axisgtd.UIDName,
				&uid.Name,
				&uid.Status)
			checkerr(err)

			if uid.Status {
				data := AxisGTDJsonType{
					Todolist: axisgtd.Todolist,
					Config:   axisgtd.Config,
					Time:     axisgtd.Time,
				}
				jsonData, _ := json.Marshal(data)

				return c.JSON(string(jsonData))
			} else {
				return c.SendStatus(404)
			}

		}
		return c.SendStatus(404)

	})

	app.Post("/sync/:name", func(c *fiber.Ctx) error {

		var exists bool
		existsQuery := `SELECT EXISTS(SELECT 1 FROM UID WHERE name = $1)`
		err = db.QueryRow(existsQuery, c.Params("name")).Scan(&exists)
		checkerr(err)
		if !exists {
			return c.SendStatus(404)
		}

		var status bool
		statusQuery := `SELECT status FROM uid WHERE name=$1`
		err = db.QueryRow(statusQuery, c.Params("name")).Scan(&status)
		checkerr(err)
		if !status {
			return c.SendStatus(404)
		}

		todo_data := new(AxisGTDType)
		checkerr(err)
		if err := c.BodyParser(todo_data); err != nil {
			return err
		}

		query := `INSERT INTO axisgtd (todolist,config,time,uid_name) VALUES ($1,$2,$3,$4)`
		_, err = db.Exec(query, todo_data.Todolist, todo_data.Config, todo_data.Time, c.Params("name"))
		checkerr(err)

		return c.SendStatus(200)
	})

	app.Delete("/delete/:name/:time", func(c *fiber.Ctx) error {
		timeVal, err := strconv.ParseInt(c.Params("time"), 10, 64)
		checkerr(err)
		err = deleteRecord(db, c.Params("name"), timeVal)
		if err != nil {
			return c.SendStatus(404)
		}
		return c.SendStatus(200)
	})

	app.Delete("/uid/:name", func(c *fiber.Ctx) error {
		err := deleteUIDAndAxisGtdByUID(db, c.Params("name"))
		if err != nil {
			return c.SendStatus(404)
		}
		return c.SendStatus(200)
	})

	err = app.Listen(":8080")
	checkerr(err)
}

func checkerr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
