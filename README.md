![logo](/img/logo.png)
---
AxisGTDSync is a sync server for [AxisGTD](https://github.com/magician333/AxisGTD), You can use this program to achieve multi-terminal synchronization of AxisGTD. Multiple IDs can achieve multi-user/multi-workspace synchronization.

**！This server is experimental and it is not recommended to use him in a production environment.**


## Technology Stack and Features
* [Golang](https://github.com/golang/go) **Ugly language**, but decent performance
* [Fiber](https://github.com/gofiber/fiber) Good performance, simple and clear
* [PostgreSQL](https://www.postgresql.org) A relatively stable SQL database for data security


## How to use(manual)
```bash
// Please make sure you already have golang running environment and enable PostgreSQL service

git clone https://github.com/magician333/AxisGTDSync.git

cd AxisGTDSync

go mod download

// Edit config.json,replace the psqlUrl and corsUrl to your config

go build -o main .

./main
```
For example, your domain is [*www.sync.app*]


> Open the browser and view [*www.sync.app*], If you see the following page, it means the service is running successfully


![success](/img/success.png)

> Open [*www.sync.app*]/create, you will get an ID

![create](/img/create.png)

> Open [*www.sync.app*]/api/docs, you can use openAPI docs (swagger) to test
> 
![swagger](/img/swaggerui.png)

> Paste the domain name and ID into the Axisgtd synchronization page and you can use it.

![syncview](/img/syncview.png)


## TodoList
- [x] Use PostgreSQL
- [x] Multi ID manage
- [x] Front-end management data page
- [x] Delete Data
- [x] Delete ID
- [x] ID status manage
- [x] Swagger API Docs
- [x] Docker
- [ ] Front-end management ID page
- [ ] Code optimization
  
## Other
You can use Firebase, Supabase, Neon and other Serverless databases to get a good experience.

We may open up the official AxisGTD sync server in the future (not necessarily happening), but of course you can develop your own sync server for your own AxisGTD sync for better privacy.