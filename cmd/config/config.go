package config

type Configuration struct {
	Eolymp   Eolymp
	Polygon  Polygon
	Telegram Telegram
	Source   string
	SpaceId  string
}

type Eolymp struct {
	ApiUrl      string
	Username    string
	Password    string
	SpaceImport string
}

type Polygon struct {
	Login    string
	Password string
}

type Telegram struct {
	Token    string
	ChatId   int64
	Problems []TelegramProblem
}

type TelegramProblem struct {
	Id   string
	Link string
	PId  string
}
