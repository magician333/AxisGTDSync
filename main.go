package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/contrib/swagger"
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

var db *sql.DB

func GetConfig() (psqlUrl string, corsUrl string) {
	configPath := "./config.json"

	content, err := os.ReadFile(configPath)
	checkerr(err)

	var config map[string]string
	err = json.Unmarshal(content, &config)
	checkerr(err)

	return config["psqlUrl"], config["corsUrl"]
}

func initDB() {

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

func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, (n+1)/2)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

func getName() (string, error) {
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

func deleteDBRecord(uidName string, time int64) error {
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

func deleteUIDAndAxisGtdByUID(uidName string) error {
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

//	@title			AxisGTD Sync API
//	@version		1.0
//	@description	API for synchronizing AxisGTD tasks and configurations.
//	@termsOfService	http://swagger.io/terms/
//	@contact.name	API Support
//	@contact.email	support@axisgtd.com
//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html
//	@host			localhost:8080
//	@BasePath		/api
//	@schemes		http

//	@securityDefinitions.apikey	APIKeyAuth
//	@in							header
//	@name						Authorization

//	@securityDefinitions.basic	BasicAuth

// @securitydefinitions.oauth2	OAuth2
// @tokenUrl					https://example.com/oauth/token
// @scope.write				Write access
// @scope.read					Read access
func main() {
	psqlUrl, corsUrl := GetConfig()

	var err error
	db, err = sql.Open("postgres", psqlUrl)
	checkerr(err)
	initDB()
	app := fiber.New()
	app.Get("/", Index)
	app.Use(swagger.New(swagger.Config{
		BasePath: "/api",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: corsUrl,
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	app.Get("/create", CreateID)

	app.Get("/id/:name", GetID)

	app.Delete("/id/:name", DeleteID)

	app.Get("/ids", GetAllID)

	app.Get("/status/:name", ToggleStatus)

	app.Get("/sync/:name", SyncGet)

	app.Post("/sync/:name", SyncPost)

	app.Delete("/delete/:name/:time", DeleteRecord)

	err = app.Listen(":8080")
	checkerr(err)
}

// @Summary		Check service status
// @Description	Checks if the AxisGTD synchronization service is running.
// @Tags			index
// @Produce		plain
// @Success		200	{string}	string	"AxisGTD synchronization service has been run successfully!"
// @Router			/ [get]
func Index(c *fiber.Ctx) error {
	return c.SendString("AxisGTD synchronization service has been run successfully!")
}

// @Summary		Create a new UID and axisgtd table
// @Description	Creates a new UID with a generated name and sets up the axisgtd table.
// @Tags			id
// @Accept			json
// @Produce		json
// @Success		200	{string}	string	"Create ID successful! Your ID is {uidName}"
// @Failure		500	{string}	string	"Internal server error"
// @Router			/create [get]
func CreateID(c *fiber.Ctx) error {
	uidName, err := getName()
	checkerr(err)

	query := `INSERT INTO UID (name, status) VALUES ($1, $2)`
	_, err = db.Exec(query, uidName, true)
	checkerr(err)

	createDataTableQuery := `
	CREATE TABLE IF NOT EXISTS axisgtd (
		todolist TEXT NOT NULL,
		config TEXT NOT NULL,
		time BIGINT NOT NULL,
		uid_name CHARACTER VARYING(100) NOT NULL,
		CONSTRAINT fk_uid_name FOREIGN KEY (uid_name) REFERENCES UID(name)
	);`

	_, err = db.Exec(createDataTableQuery)
	checkerr(err)

	msg := fmt.Sprintf("Create ID successful! Your ID is %s", uidName)
	return c.SendString(msg)
}

// @Summary		Get AxisGTD records by UID name
// @Description	Retrieves a list of AxisGTD records associated with the given UID name.
// @Tags			id
// @Accept			json
// @Produce		json
// @Param			name	path		string	true	"UID Name"
// @Success		200		{array}		AxisGTDJsonType
// @Failure		404		{string}	string	"No records found"
// @Failure		500		{string}	string	"Internal server error"
// @Router			/id/{name} [get]
func GetID(c *fiber.Ctx) error {
	query := `
		SELECT
			axisgtd.todolist, 
			axisgtd.config, 
			axisgtd.time, 
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
		var status bool
		err := rows.Scan(&axisgtd.Todolist, &axisgtd.Config, &axisgtd.Time, &status)
		checkerr(err)
		if status {
			dataList = append(dataList, AxisGTDJsonType{
				Todolist: axisgtd.Todolist,
				Config:   axisgtd.Config,
				Time:     axisgtd.Time,
			})
		}
	}

	if len(dataList) == 0 {
		return c.SendStatus(404)
	}

	return c.JSON(dataList)
}

// @Summary		Delete a UID and associated axisgtd records
// @Description	Deletes a UID and all associated axisgtd records from the database.
// @Tags			id
// @Accept			json
// @Produce		json
// @Param			name	path		string	true	"UID Name"
// @Success		200		{string}	string	"UID and associated records deleted successfully"
// @Failure		500		{string}	string	"Internal server error"
// @Router			/id/{name} [delete]
func DeleteID(c *fiber.Ctx) error {
	err := deleteUIDAndAxisGtdByUID(c.Params("name"))
	checkerr(err)
	return c.SendStatus(200)
}

// @Summary		Get counts of axisgtd per UID
// @Description	Retrieves the count of axisgtd entries associated with each UID.
// @Tags			id
// @Accept			json
// @Produce		json
// @Success		200	{array}		IDSType
// @Failure		500	{string}	string	"Internal server error"
// @Router			/ids [get]
func GetAllID(c *fiber.Ctx) error {
	query := `
		SELECT
			UID.name,
			UID.status,
			COUNT(axisgtd.uid_name) AS axisgtd_count
		FROM
			UID
		LEFT JOIN axisgtd ON UID.name = axisgtd.uid_name
		GROUP BY
			UID.name, UID.status`
	rows, err := db.Query(query)
	checkerr(err)
	defer rows.Close()

	var ids []IDSType
	for rows.Next() {
		var preID IDSType
		err := rows.Scan(&preID.Name, &preID.Status, &preID.Count)
		checkerr(err)
		ids = append(ids, preID)
	}

	return c.JSON(ids)
}

// @Summary		Toggle the status of a UID
// @Description	Updates the status field of a UID to the opposite value.
// @Tags			status
// @Accept			json
// @Produce		json
// @Param			name	path		string	true	"UID Name"
// @Success		200		{object}	string	"Status toggled successfully"
// @Failure		404		{string}	string	"UID not found"
// @Failure		500		{string}	string	"Internal server error"
// @Router			/status/{name} [get]
func ToggleStatus(c *fiber.Ctx) error {
	var uid UID
	searchQuery := `SELECT name, status FROM UID WHERE name = $1`
	err := db.QueryRow(searchQuery, c.Params("name")).Scan(&uid.Name, &uid.Status)
	if err != nil {
		return c.SendStatus(404)
	}

	uid.Status = !uid.Status

	updateQuery := `UPDATE UID SET status = $1 WHERE name = $2`
	_, err = db.Exec(updateQuery, uid.Status, c.Params("name"))
	checkerr(err)
	return c.JSON(fiber.Map{"message": "Status toggled", "new_status": uid.Status})
}

// @Summary		Get the latest AxisGTD record by UID name
// @Description	Retrieves the latest AxisGTD record associated with the specified UID name, ordered by time in descending order.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Param			name	path		string			true	"UID Name"
// @Success		200		{object}	AxisGTDJsonType	"The latest AxisGTD record"
// @Failure		404		{string}	string			"UID not found or no records available"
// @Failure		500		{string}	string			"Internal server error"
// @Router			/sync/{name} [get]
func SyncGet(c *fiber.Ctx) error {
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
			return c.JSON(data)
		} else {
			return c.SendStatus(404)
		}

	}
	return c.SendStatus(404)

}

// @Summary		Create a new AxisGTD record
// @Description	Inserts a new AxisGTD record into the database for the given UID name.
// @Tags			sync
// @Accept			json
// @Produce		json
// @Param			name		path		string		true	"UID Name"
// @Param			todo_data	body		AxisGTDType	true	"AxisGTD record to create"
// @Success		200			{string}	string		"Record created successfully"
// @Failure		404			{string}	string		"UID not found or UID is disabled"
// @Failure		400			{string}	string		"Invalid request body"
// @Failure		500			{string}	string		"Internal server error"
// @Router			/sync/{name} [post]
func SyncPost(c *fiber.Ctx) error {

	var exists bool
	existsQuery := `SELECT EXISTS(SELECT 1 FROM UID WHERE name = $1)`
	err := db.QueryRow(existsQuery, c.Params("name")).Scan(&exists)
	checkerr(err)
	if !exists {
		return c.SendStatus(404)
	}

	var status bool
	statusQuery := `SELECT status FROM uid WHERE name=$1`
	statusErr := db.QueryRow(statusQuery, c.Params("name")).Scan(&status)
	checkerr(statusErr)
	if !status {
		return c.SendStatus(404)
	}

	todo_data := new(AxisGTDType)
	if err := c.BodyParser(todo_data); err != nil {
		return err
	}

	query := `INSERT INTO axisgtd (todolist,config,time,uid_name) VALUES ($1,$2,$3,$4)`
	_, err = db.Exec(query, todo_data.Todolist, todo_data.Config, todo_data.Time, c.Params("name"))
	checkerr(err)

	return c.SendStatus(200)
}

// @Summary		Delete a record by UID name and time
// @Description	Deletes a record from the database based on UID name and time.
// @Tags			delete
// @Accept			json
// @Produce		json
// @Param			name	path		string	true	"UID Name"
// @Param			time	path		int		true	"The record's time"
// @Success		200		{string}	string	"Record deleted successfully"
// @Failure		404		{string}	string	"Record not found"
// @Failure		500		{string}	string	"Internal server error"
// @Router			/delete/{name}/{time} [delete]
func DeleteRecord(c *fiber.Ctx) error {
	timeVal, err := strconv.ParseInt(c.Params("time"), 10, 64)
	checkerr(err)
	err = deleteDBRecord(c.Params("name"), timeVal)
	if err != nil {
		return c.SendStatus(404)
	}
	return c.SendStatus(200)
}
