package api

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

type IDSType struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
	Count  int    `json:"count"`
}

type ConfigType struct {
	PSQLURL string `json:"psql"`
	CorsURL string `json:"cors"`
}
