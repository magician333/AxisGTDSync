package api

import (
	"database/sql"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

var db *sql.DB

func init() {
	psqlUrl := GetConfig().PSQLURL
	var err error
	db, err = sql.Open("postgres", psqlUrl)
	checkerr(err)
}

// @Summary		Check service status
// @Description	Checks if the AxisGTD synchronization service is running.
// @Tags			index
// @Produce		plain
// @Success		200	{string}	string "HTML template with service status"
// @Router			/ [get]
func Index(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{"Title": "AxisGTDSync Manage"})
}

// @Summary		Create a new UID and axisgtd table
// @Description	Creates a new UID with a generated name and sets up the axisgtd table.
// @Tags			id
// @Accept			json
// @Produce		json
// @Success		200	{string}	string	"Create ID successful! Your ID is {uidName}"
// @Failure		500	{string}	string	"Internal server error"
// @Router			/create [put]
func CreateID(c *fiber.Ctx) error {
	uidName, err := GetName()
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Create ID Failed"})
	}

	query := `INSERT INTO UID (name, status) VALUES ($1, $2)`
	_, err = db.Exec(query, uidName, true)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Create ID Failed"})
	}

	createDataTableQuery := `
	CREATE TABLE IF NOT EXISTS axisgtd (
		todolist TEXT NOT NULL,
		config TEXT NOT NULL,
		time BIGINT NOT NULL,
		uid_name CHARACTER VARYING(100) NOT NULL,
		CONSTRAINT fk_uid_name FOREIGN KEY (uid_name) REFERENCES UID(name)
	);`

	_, err = db.Exec(createDataTableQuery)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Create ID Failed"})
	}

	return c.Status(200).JSON(fiber.Map{"name": uidName})
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
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Get ID information Failed"})
	}
	defer rows.Close()

	var dataList []AxisGTDJsonType

	for rows.Next() {
		var axisgtd AxisGTDType
		var status bool
		err := rows.Scan(&axisgtd.Todolist, &axisgtd.Config, &axisgtd.Time, &status)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"Error": "Get ID information Failed"})
		}
		if status {
			dataList = append(dataList, AxisGTDJsonType{
				Todolist: axisgtd.Todolist,
				Config:   axisgtd.Config,
				Time:     axisgtd.Time,
			})
		}
	}

	if len(dataList) == 0 {
		return c.Status(404).JSON(fiber.Map{"Error": "No records found"})
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
	err := DeleteUIDAndAxisGtdByUID(c.Params("name"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Delete ID Error"})
	}
	return c.Status(200).JSON(fiber.Map{"Success": "ID and associated records deleted successfully"})
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
			UID.id,
			UID.name,
			UID.status,
			COUNT(axisgtd.uid_name) AS axisgtd_count
		FROM
			UID
		LEFT JOIN axisgtd ON UID.name = axisgtd.uid_name
		GROUP BY
			UID.id,UID.name, UID.status`
	rows, err := db.Query(query)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Get ID list Failed"})
	}
	defer rows.Close()

	var ids []IDSType
	for rows.Next() {
		var preID IDSType
		err := rows.Scan(&preID.Id, &preID.Name, &preID.Status, &preID.Count)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"Error": "Get ID list Failed"})
		}
		ids = append(ids, preID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].Id < ids[j].Id
	})
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
		return c.Status(404).JSON(fiber.Map{"Error": "UID not found"})
	}

	uid.Status = !uid.Status

	updateQuery := `UPDATE UID SET status = $1 WHERE name = $2`
	_, err = db.Exec(updateQuery, uid.Status, c.Params("name"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Change Status Failed"})
	}
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
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Get sync data Failed"})
	}
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
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"Error": "Get sync data Failed"})
		}
		if uid.Status {
			data := AxisGTDJsonType{
				Todolist: axisgtd.Todolist,
				Config:   axisgtd.Config,
				Time:     axisgtd.Time,
			}
			return c.JSON(data)
		} else {
			return c.Status(404).JSON(fiber.Map{"Error": c.Params("name") + " not found"})
		}

	}
	return c.Status(404).JSON(fiber.Map{"Error": "No records available"})

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
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Post sync data Failed"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"Error": c.Params("name") + " not found"})
	}

	var status bool
	statusQuery := `SELECT status FROM uid WHERE name=$1`
	err = db.QueryRow(statusQuery, c.Params("name")).Scan(&status)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Post sync data Failed"})
	}
	if !status {
		return c.Status(404).JSON(fiber.Map{"Error": c.Params("name") + " is disabled"})
	}

	todo_data := new(AxisGTDType)
	if err := c.BodyParser(todo_data); err != nil {
		return err
	}

	query := `INSERT INTO axisgtd (todolist,config,time,uid_name) VALUES ($1,$2,$3,$4)`
	_, err = db.Exec(query, todo_data.Todolist, todo_data.Config, todo_data.Time, c.Params("name"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Post sync data Failed"})
	}

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
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Delete Record Failed"})
	}
	err = DeleteDBRecord(c.Params("name"), timeVal)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"Error": "Record not found"})
	}
	return c.SendStatus(200)
}
