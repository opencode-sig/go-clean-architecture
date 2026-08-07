// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration values loaded from YAML and defaults.
type Config struct {
	ServerPort      string
	ServerReadHdrT  time.Duration
	ServerReadT     time.Duration
	ServerWriteT    time.Duration
	ServerIdleT     time.Duration
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	DBCharset       string
	DBParsetime     string
	DBLoc           string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBConnMaxLive   time.Duration
	CacheEnabled    bool
	CacheType       string
	CacheTTL        time.Duration
	RedisHost       string
	RedisPort       int
	RedisDB         int
	RedisPass       string
	RateLimitRPS    float64
	RateLimitBurst  int
	DefaultPageSize int
	MaxPageSize     int
	LogLevel        string
	LogFile         string
	LogMaxSize      int
	LogMaxAge       int
	LogMaxBK        int
}

// DSN builds a MySQL connection string from Config fields.
func (c *Config) DSN() string {
	loc := c.DBLoc
	if loc == "" {
		loc = "Local"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
		c.DBCharset, c.DBParsetime, loc,
	)
}

type configFile struct {
	Server struct {
		Port     string `yaml:"port"`
		ReadHdrT int    `yaml:"read_header_timeout"`
		ReadT    int    `yaml:"read_timeout"`
		WriteT   int    `yaml:"write_timeout"`
		IdleT    int    `yaml:"idle_timeout"`
	} `yaml:"server"`
	MySQL struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		User         string `yaml:"user"`
		Password     string `yaml:"password"`
		Name         string `yaml:"name"`
		Charset      string `yaml:"charset"`
		Parsetime    string `yaml:"parse_time"`
		Loc          string `yaml:"loc"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		ConnMaxLife  int    `yaml:"conn_max_lifetime"`
	} `yaml:"mysql"`
	Cache struct {
		Enabled *bool  `yaml:"enabled"` // global cache switch; nil means enabled
		Type    string `yaml:"type"`    // "memory" or "redis"
		TTL     int    `yaml:"ttl"`     // default TTL in seconds
	} `yaml:"cache"`
	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		DB       int    `yaml:"db"`
		Password string `yaml:"password"`
	} `yaml:"redis"`
	RateLimit struct {
		RPS   float64 `yaml:"rps"`
		Burst int     `yaml:"burst"`
	} `yaml:"rate_limit"`
	Pagination struct {
		DefaultPageSize int `yaml:"default_page_size"`
		MaxPageSize     int `yaml:"max_page_size"`
	} `yaml:"pagination"`
	Log struct {
		Level   string `yaml:"level"`
		File    string `yaml:"file"`
		MaxSize int    `yaml:"max_size"`
		MaxAge  int    `yaml:"max_age"`
		MaxBK   int    `yaml:"max_backups"`
	} `yaml:"log"`
}

// LoadConfig returns a Config populated with defaults merged from the given YAML file.
// If configPath is empty, it tries config.yaml first, then config/development.yaml.
func LoadConfig(configPath string) *Config {
	defaults := &Config{
		ServerPort:      "8080",
		ServerReadHdrT:  5 * time.Second,
		ServerReadT:     30 * time.Second,
		ServerWriteT:    30 * time.Second,
		ServerIdleT:     120 * time.Second,
		DBHost:          "127.0.0.1",
		DBPort:          3306,
		DBUser:          "root",
		DBPassword:      "password",
		DBName:          "zhisuo",
		DBCharset:       "utf8mb4",
		DBParsetime:     "True",
		DBLoc:           "Local",
		DBMaxOpenConns:  25,
		DBMaxIdleConns:  5,
		DBConnMaxLive:   5 * time.Minute,
		CacheEnabled:    true,
		CacheType:       "memory",
		CacheTTL:        5 * time.Minute,
		RedisHost:       "127.0.0.1",
		RedisPort:       6379,
		RateLimitRPS:    10,
		RateLimitBurst:  50,
		DefaultPageSize: 20,
		MaxPageSize:     100,
		LogLevel:        "info",
		LogFile:         "logs/app.log",
		LogMaxSize:      100,
		LogMaxAge:       30,
		LogMaxBK:        7,
	}

	paths := []string{configPath}
	if configPath == "" {
		paths = []string{"config.yaml", "config/development.yaml"}
	}

	for _, p := range paths {
		if p == "" {
			continue
		}

		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()

		var cf configFile
		if yaml.NewDecoder(f).Decode(&cf) != nil {
			continue
		}

		applyFile(defaults, &cf)
		return defaults
	}

	return defaults
}

func applyFile(cfg *Config, cf *configFile) {
	if cf.Server.Port != "" {
		cfg.ServerPort = cf.Server.Port
	}
	if cf.Server.ReadHdrT > 0 {
		cfg.ServerReadHdrT = time.Duration(cf.Server.ReadHdrT) * time.Second
	}
	if cf.Server.ReadT > 0 {
		cfg.ServerReadT = time.Duration(cf.Server.ReadT) * time.Second
	}
	if cf.Server.WriteT > 0 {
		cfg.ServerWriteT = time.Duration(cf.Server.WriteT) * time.Second
	}
	if cf.Server.IdleT > 0 {
		cfg.ServerIdleT = time.Duration(cf.Server.IdleT) * time.Second
	}
	if cf.MySQL.Host != "" {
		cfg.DBHost = cf.MySQL.Host
	}
	if cf.MySQL.Port > 0 {
		cfg.DBPort = cf.MySQL.Port
	}
	if cf.MySQL.User != "" {
		cfg.DBUser = cf.MySQL.User
	}
	if cf.MySQL.Password != "" {
		cfg.DBPassword = cf.MySQL.Password
	}
	if cf.MySQL.Name != "" {
		cfg.DBName = cf.MySQL.Name
	}
	if cf.MySQL.Charset != "" {
		cfg.DBCharset = cf.MySQL.Charset
	}
	if cf.MySQL.Parsetime != "" {
		cfg.DBParsetime = cf.MySQL.Parsetime
	}
	if cf.MySQL.Loc != "" {
		cfg.DBLoc = cf.MySQL.Loc
	}
	if cf.MySQL.MaxOpenConns > 0 {
		cfg.DBMaxOpenConns = cf.MySQL.MaxOpenConns
	}
	if cf.MySQL.MaxIdleConns > 0 {
		cfg.DBMaxIdleConns = cf.MySQL.MaxIdleConns
	}
	if cf.MySQL.ConnMaxLife > 0 {
		cfg.DBConnMaxLive = time.Duration(cf.MySQL.ConnMaxLife) * time.Second
	}
	if cf.Cache.Type != "" {
		cfg.CacheType = cf.Cache.Type
	}
	if cf.Cache.Enabled != nil {
		cfg.CacheEnabled = *cf.Cache.Enabled
	} else {
		cfg.CacheEnabled = true
	}
	if cf.Cache.TTL > 0 {
		cfg.CacheTTL = time.Duration(cf.Cache.TTL) * time.Second
	}
	if cf.Redis.Host != "" {
		cfg.RedisHost = cf.Redis.Host
	}
	if cf.Redis.Port > 0 {
		cfg.RedisPort = cf.Redis.Port
	}
	if cf.Redis.DB > 0 {
		cfg.RedisDB = cf.Redis.DB
	}
	if cf.Redis.Password != "" {
		cfg.RedisPass = cf.Redis.Password
	}
	if cf.RateLimit.RPS > 0 {
		cfg.RateLimitRPS = cf.RateLimit.RPS
	}
	if cf.RateLimit.Burst > 0 {
		cfg.RateLimitBurst = cf.RateLimit.Burst
	}
	if cf.Pagination.DefaultPageSize > 0 {
		cfg.DefaultPageSize = cf.Pagination.DefaultPageSize
	}
	if cf.Pagination.MaxPageSize > 0 {
		cfg.MaxPageSize = cf.Pagination.MaxPageSize
	}
	if cf.Log.Level != "" {
		cfg.LogLevel = cf.Log.Level
	}
	if cf.Log.File != "" {
		cfg.LogFile = cf.Log.File
	}
	if cf.Log.MaxSize > 0 {
		cfg.LogMaxSize = cf.Log.MaxSize
	}
	if cf.Log.MaxAge > 0 {
		cfg.LogMaxAge = cf.Log.MaxAge
	}
	if cf.Log.MaxBK > 0 {
		cfg.LogMaxBK = cf.Log.MaxBK
	}
}
