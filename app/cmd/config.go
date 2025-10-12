package cmd

import (
	"fmt"
	"github.com/go-bumbu/config"
	"github.com/gorilla/securecookie"
	"strconv"
)

type AppCfg struct {
	Server serverCfg
	Obs    serverCfg `config:"Observability"`
	Auth   authConfig
	Env    Env
	Msgs   []Msg
}

type Env struct {
	LogLevel   string
	Production bool
}

type serverCfg struct {
	BindIp string
	Port   int
}

func (c serverCfg) Addr() string {
	if c.BindIp == "" {
		return ":" + strconv.Itoa(c.Port)
	}
	return c.BindIp + ":" + strconv.Itoa(c.Port)
}

type authConfig struct {
	SessionPath   string
	HashKeyBytes  []byte
	BlockKeyBytes []byte
	HashKey       string
	BlockKey      string
	UserStore     userStore
}

type userStore struct {
	StoreType string `config:"Type"` // can be static | file
	FilePath  string `config:"Path"`
	Users     []User
}
type User struct {
	Name string
	Pw   string
}

// Default represents the basic set of sensible defaults
var defaultCfg = AppCfg{

	Server: serverCfg{
		BindIp: "",
		Port:   8085,
	},
	Obs: serverCfg{
		BindIp: "",
		Port:   9090,
	},
	Auth: authConfig{
		SessionPath: "", // location where the sessions are stored
		HashKey:     "", // cookie store encryption key
		BlockKey:    "", // cookie value encryption
		UserStore: userStore{
			StoreType: "static",
			Users: []User{
				{
					Name: "demo",
					Pw:   "demo",
				},
				{
					Name: "admin",
					Pw:   "admin",
				},
			},
		},
	},
	Env: Env{
		LogLevel:   "info",
		Production: true,
	},
}

type Msg struct {
	Level string
	Msg   string
}

func getAppCfg(file string) (AppCfg, error) {
	configMsg := []Msg{}
	cfg := AppCfg{}
	var err error
	_, err = config.Load(
		config.Defaults{Item: defaultCfg},
		config.CfgFile{Path: file, Mandatory: false},
		config.EnvVar{Prefix: "BUMBU"},
		config.Unmarshal{Item: &cfg},
		config.Writer{Fn: func(level, msg string) {
			if level == config.InfoLevel {
				configMsg = append(configMsg, Msg{Level: "info", Msg: msg})
			}
			if level == config.DebugLevel {
				configMsg = append(configMsg, Msg{Level: "debug", Msg: msg})
			}
		}},
	)
	cfg.Msgs = configMsg
	if err != nil {
		return cfg, err
	}

	// handle auth cookie hash
	if cfg.Auth.HashKey == "" || cfg.Auth.BlockKey == "" {
		cfg.Auth.HashKeyBytes = securecookie.GenerateRandomKey(64)
		cfg.Auth.BlockKeyBytes = securecookie.GenerateRandomKey(32)
	}

	if len(cfg.Auth.HashKey) != 64 {
		return cfg, fmt.Errorf("hashkey must be 64 chars long")
	}
	if len(cfg.Auth.BlockKey) != 32 {
		return cfg, fmt.Errorf("blockKey must be 32 chars long")
	}

	cfg.Auth.HashKeyBytes = []byte(cfg.Auth.HashKey)
	cfg.Auth.BlockKeyBytes = []byte(cfg.Auth.BlockKey)

	return cfg, nil
}
