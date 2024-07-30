package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type DataType struct {
	Todolist string
	Config   string
	Time     int
}

func main() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000/",
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("index")
	})

	app.Get("/sync", func(c *fiber.Ctx) error {
		f, err := os.Open("Data.json")
		if err != nil {
			return c.SendString("")
		}
		defer f.Close()
		bytes, _ := io.ReadAll(f)
		var rdata DataType
		err = json.Unmarshal(bytes, &rdata)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			return c.SendString("")
		}
		return c.JSON(rdata)
	})

	app.Post("/sync", func(c *fiber.Ctx) error {
		todo_data := new(DataType)
		if err := c.BodyParser(todo_data); err != nil {
			return err
		}

		data := DataType{
			Todolist: todo_data.Todolist,
			Config:   todo_data.Config,
			Time:     todo_data.Time,
		}
		file, err := os.Create("Data.json")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return err
		}
		defer file.Close()
		jsonData, _ := json.MarshalIndent(data, "", "    ")
		_, err = file.Write(jsonData)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return err
		}
		return c.SendString(todo_data.Todolist)
	})

	app.Listen(":8080")
}
