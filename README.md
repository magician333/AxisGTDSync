![logo](/img/logo.png)
---
AxisGTDSync is a sync server for [AxisGTD](https://github.com/magician333/AxisGTD), You can use this program to achieve multi-terminal synchronization of AxisGTD. Multiple IDs can achieve multi-user/multi-workspace synchronization.

**！This server is experimental and it is not recommended to use him in a production environment.**


## Technology Stack and Features
* [Golang](https://github.com/golang/go) **Ugly language**, but decent performance
* [Fiber](https://github.com/gofiber/fiber) Good performance, simple and clear
* [PostgreSQL](https://www.postgresql.org) A relatively stable SQL database for data security


## How to use
```bash
// Please make sure you already have golang running environment and enable PostgreSQL service
git clone git@github.com:magician333/AxisGTDSync.git
cd AxisGTDSync

// Edit config.json,replace the psqlUrl and corsUrl to your config
go build main.go
./main
```
For example, your domain name is **www.sync.app**

Open the browser and view www.axisgtdsync.app, If you see the following page, it means the service is running successfully

![success](/img/success.png)

Open www.sync.app/create, you will get an ID

![create](/img/create.png)

Open www.sync.app/docs, you can use openAPI docs (swagger) to test
![swagger](/img/swaggerui.png)

paste the domain name and ID into the Axisgtd synchronization page and you can use it.

![syncview](/img/syncview.png)


## TodoList
- [x] Use PostgreSQL
- [x] Multi ID manage
- [x] Front-end management data page
- [x] Delete Data
- [x] Delete ID
- [x] ID status manage
- [x] Swagger API Docs
- [ ] Front-end management ID page
- [ ] Code optimization
- [ ] Docker
  
## Other
You can use Firebase, Supabase, Neon and other Serverless databases to get a good experience.

We may open up the official AxisGTD sync server in the future (not necessarily happening), but of course you can develop your own sync server for your own AxisGTD sync for better privacy.