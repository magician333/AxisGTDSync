package main

import (
	"AxisGTDSync/api"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

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

	corsUrl := api.GetConfig().CorsURL

	api.InitDB()

	app := fiber.New()
	app.Get("/", api.Index)
	app.Use(swagger.New(swagger.Config{
		BasePath: "/api",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: corsUrl,
		AllowHeaders: "Origin,Content-Type,Accept",
	}))

	app.Get("/create", api.CreateID)

	app.Get("/id/:name", api.GetID)

	app.Delete("/id/:name", api.DeleteID)

	app.Get("/ids", api.GetAllID)

	app.Get("/status/:name", api.ToggleStatus)

	app.Get("/sync/:name", api.SyncGet)

	app.Post("/sync/:name", api.SyncPost)

	app.Delete("/delete/:name/:time", api.DeleteRecord)

	app.Listen(":8080")
}
