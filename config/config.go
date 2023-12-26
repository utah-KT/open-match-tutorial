package config

import "github.com/BurntSushi/toml"

var (
	Global Config
	Path   = "/etc/open-match-tutorial/config.toml"
)

type OpenMatch struct {
	FrontendEndpoint string `toml:"frontend_endpoint"`
	BackendEndpoint  string `toml:"backend_endpoint"`
	QueryEndpoint    string `toml:"query_endpoint"`
}

type Matching struct {
	Tag string `toml:"tag"`
}

type GameFront struct {
	Port int `toml:"port"`
}

type GameServer struct {
	MemberNum int    `toml:"member_num"`
	Timeout   int64  `toml:"timeout"`
	Endpoint  string `toml:"endpoint"`
}

type Mmf struct {
	Port int    `toml:"port"`
	Name string `toml:"name"`
	Host string `toml:"host"`
}

type Config struct {
	OpenMatch  OpenMatch  `toml:"open_match"`
	Matching   Matching   `toml:"matching"`
	GameFront  GameFront  `toml:"gamefront"`
	GameServer GameServer `toml:"gameserver"`
	Mmf        Mmf        `toml:"mmf"`
}

func Load() {
	conf := &Config{}
	_, err := toml.DecodeFile(Path, conf)
	if err != nil {
		panic(err)
	}
	Global = *conf
}
