package config

type Config struct {
	PgConnStr string
	Addr      string
}

func NewConfig(pgConnStr string, addr string) *Config {
	return &Config{PgConnStr: pgConnStr,
		Addr: addr}
}
