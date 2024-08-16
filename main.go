package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/lib/pq"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type DataType struct {
	Todolist string
	Config   string
	Time     int
}

func main() {
	psqlURL := "user='youruser' password='yourpassword' dbname='yourdbname' sslmode='disable'" // set your own postgreSQL url
	db, err := sql.Open("postgres", psqlURL)
	checkerr(err)
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS axisgtd (
        todolist TEXT NOT NULL,
        config TEXT NOT NULL,
        time BIGINT NOT NULL
    );`
	_, err = db.Exec(createTableQuery)
	checkerr(err)

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000/", //Custom set your axisgtd service url
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("AxisGTD Sync")
	})

	app.Post("/sync", func(c *fiber.Ctx) error {
		todo_data := new(DataType)
		checkerr(err)
		if err := c.BodyParser(todo_data); err != nil {
			return err
		}
		stmt, err := db.Prepare("INSERT INTO axisgtd (todolist,config,time) VALUES ($1,$2,$3)")
		checkerr(err)
		defer stmt.Close()
		_, err = stmt.Exec(todo_data.Todolist, todo_data.Config, todo_data.Time)
		checkerr(err)
		return c.SendString("aaa")
	})

	app.Get("/sync", func(c *fiber.Ctx) error {
		stmt, err := db.Query("SELECT * FROM axisgtd LIMIT 1")
		checkerr(err)
		for stmt.Next() {
			var todolist string
			var config string
			var time int
			err = stmt.Scan(&todolist, &config, &time)
			checkerr(err)
			data := DataType{
				Todolist: todolist,
				Config:   config,
				Time:     time,
			}
			jsonData, _ := json.MarshalIndent(data, "", "    ")
			return c.SendString(string(jsonData))
		}
		return c.SendString("")
	})

	// TODOList
	// Manage sql data

	app.Listen(":8080")
}

func checkerr(err error) {

	if err != nil {
		log.Fatal(err)
	}
}
