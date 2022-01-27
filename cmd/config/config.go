package config

type Configuration struct {
	Eolymp  Eolymp
	Polygon Polygon
	Source  string
	SpaceId string
}

type Eolymp struct {
	ApiUrl   string
	Username string
	Password string
}

type Polygon struct {
	Login    string
	Password string
}
