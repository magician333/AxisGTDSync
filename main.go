package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	_ "github.com/lib/pq"
)

type DataType struct {
	Todolist string `json:"todolist"`
	Config   string `json:"config"`
	Time     int64  `json:"time"`
	UIDName  string `json:"uidname"`
}
type UID struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

type DataJsonType struct {
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

type Config struct {
	psqlUrl string
	corsUrl string
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

func GetConfig() (psqlUrl string, corsUrl string) {
	configPath := "./config.json"
	// file, err := os.Open(configPath)
	// checkerr(err)
	// defer file.Close()

	content, err := ioutil.ReadFile(configPath)
	checkerr(err)

	var config map[string]string
	err = json.Unmarshal(content, &config)
	checkerr(err)

	return config["psqlUrl"], config["corsUrl"]
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

		stmt, err := db.Prepare("INSERT INTO UID (name,status) VALUES ($1,$2)")
		checkerr(err)
		defer stmt.Close()
		_, err = stmt.Exec(uidName, true)
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

		var dataList []DataJsonType

		for rows.Next() {
			var axisgtd DataType
			var uid UID
			err := rows.Scan(&axisgtd.Todolist,
				&axisgtd.Config,
				&axisgtd.Time,
				&uid.Name,
				&uid.Status)
			checkerr(err)
			if uid.Status {
				dataList = append(dataList, DataJsonType{
					Todolist: axisgtd.Todolist,
					Config:   axisgtd.Config,
					Time:     axisgtd.Time,
				})
			}
			if len(dataList) == 0 {
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

	app.Post("/sync/:name", func(c *fiber.Ctx) error {

		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM UID WHERE name = $1)"
		err = db.QueryRow(query, c.Params("name")).Scan(&exists)
		checkerr(err)
		if !exists {
			return c.SendStatus(404)
		}

		todo_data := new(DataType)
		checkerr(err)
		if err := c.BodyParser(todo_data); err != nil {
			return err
		}
		stmt, err := db.Prepare("INSERT INTO axisgtd (todolist,config,time,uid_name) VALUES ($1,$2,$3,$4)")
		checkerr(err)
		defer stmt.Close()
		_, err = stmt.Exec(todo_data.Todolist, todo_data.Config, todo_data.Time, c.Params("name"))
		checkerr(err)

		return c.SendStatus(200)
	})

	app.Get("/sync/:name", func(c *fiber.Ctx) error {

		rows, err := db.Query(`
        SELECT 
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
		LIMIT 1;
    `, c.Params("name"))

		checkerr(err)
		defer rows.Close()
		for rows.Next() {
			var axisgtd DataType
			var uid UID
			err := rows.Scan(&axisgtd.Todolist,
				&axisgtd.Config,
				&axisgtd.Time,
				&axisgtd.UIDName,
				&uid.Name,
				&uid.Status)
			checkerr(err)

			if uid.Status {
				data := DataJsonType{
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

	app.Get("/delete/:name/:time", func(c *fiber.Ctx) error {
		//Ready Todo
		return c.SendString(c.Params("time"))
	})

	err = app.Listen(":8080")
	checkerr(err)
}

func checkerr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
