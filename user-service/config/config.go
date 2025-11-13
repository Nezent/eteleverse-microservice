package config

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Root Config struct
type Config struct {
	App       AppConfig      `mapstructure:"app"`
	Database  DatabaseConfig `mapstructure:"database"`
	Redis     RedisConfig    `mapstructure:"redis"`
	Log       LogConfig      `mapstructure:"log"`
	AdminAuth AuthConfig     `mapstructure:"admin_auth"`
	Auth      AuthConfig     `mapstructure:"auth"`
}

// -------------------- App --------------------

type AppConfig struct {
	Name        string            `mapstructure:"name"`
	Version     string            `mapstructure:"version"`
	Description string            `mapstructure:"description"`
	Env         string            `mapstructure:"env"`
	Debug       bool              `mapstructure:"debug"`
	Timezone    string            `mapstructure:"timezone"`
	URL         string            `mapstructure:"url"`
	FrontendURL string            `mapstructure:"frontend_url"`
	AssetsURL   string            `mapstructure:"assets_url"`
	Host        string            `mapstructure:"host"`
	Port        int               `mapstructure:"port"`
	Maintenance MaintenanceConfig `mapstructure:"maintenance"`
}

type MaintenanceConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Message string `mapstructure:"message"`
	Driver  string `mapstructure:"driver"`
}

// -------------------- Database --------------------

type DatabaseConfig struct {
	Default  string           `mapstructure:"default"`
	MySQL    DBInstanceConfig `mapstructure:"mysql"`
	Postgres DBInstanceConfig `mapstructure:"postgres"`
	TiDB     DBInstanceConfig `mapstructure:"tidb"`
}

func (d *DatabaseConfig) BuildDsn() string {
	conn := d.Driver()

	switch conn.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			conn.User, conn.Password, conn.Host, conn.Port, conn.Name)

	case "pgsql", "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
			conn.Host, conn.Port, conn.User, conn.Name, conn.Password)
	}
	return ""
}

func (d *DatabaseConfig) Driver() DBInstanceConfig {
	switch d.Default {
	case "mysql":
		return d.MySQL
	case "postgres", "postgresql":
		return d.Postgres
	case "tidb":
		return d.TiDB
	default:
		return DBInstanceConfig{
			Driver: d.Default,
		}
	}
}

type DBInstanceConfig struct {
	Driver        string     `mapstructure:"driver"`
	Host          string     `mapstructure:"host"`
	Port          int        `mapstructure:"port"`
	User          string     `mapstructure:"user"`
	Password      string     `mapstructure:"password"`
	Name          string     `mapstructure:"name"`
	MigrationPath string     `mapstructure:"migration_path"`
	Pool          PoolConfig `mapstructure:"pool"`
}

type PoolConfig struct {
	MaxIdleConns    int           `mapstructure:"max_idle_connections"`
	MaxOpenConns    int           `mapstructure:"max_open_connections"`
	ConnMaxLifetime time.Duration `mapstructure:"max_connection_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Default  string `mapstructure:"default"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LogConfig struct {
	Level             string            `mapstructure:"level"`
	Format            string            `mapstructure:"format"`
	TimeEncoding      string            `mapstructure:"time_encoding"`
	File              LogFileConfig     `mapstructure:"file"`
	Sampling          LogSamplingConfig `mapstructure:"sampling"`
	DisableTimestamp  bool              `mapstructure:"disable_timestamp"`
	DisableCaller     bool              `mapstructure:"disable_caller"`
	DisableStacktrace bool              `mapstructure:"disable_stacktrace"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxDays    int    `mapstructure:"max_days"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
	LocalTime  bool   `mapstructure:"localtime"`
}

type LogSamplingConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Initial    int           `mapstructure:"initial"`
	Thereafter int           `mapstructure:"thereafter"`
	Tick       time.Duration `mapstructure:"tick"`
}

func (l *LogConfig) Validate() error {
	if l.Level == "" || !slices.Contains([]string{"debug", "info", "warn", "error", "fatal"}, l.Level) {
		return fmt.Errorf("invalid log level: %s", l.Level)
	}
	if l.Format == "" || !slices.Contains([]string{"text", "json"}, l.Format) {
		return fmt.Errorf("invalid log format: %s", l.Format)
	}
	if l.TimeEncoding != "" && !slices.Contains([]string{"iso8601", "epoch", "epoch_millis", "epoch_nanos"}, l.TimeEncoding) {
		return fmt.Errorf("invalid time_encoding: %s", l.TimeEncoding)
	}
	if l.File.Path != "" && !strings.HasSuffix(l.File.Path, ".log") {
		return fmt.Errorf("log file path must end with .log: %s", l.File.Path)
	}
	if l.File.MaxSize < 0 || l.File.MaxDays < 0 || l.File.MaxBackups < 0 {
		return fmt.Errorf("log file max_size, max_days, and max_backups must be non-negative")
	}
	if l.Sampling.Initial < 0 || l.Sampling.Thereafter < 0 {
		return fmt.Errorf("sampling initial/thereafter must be non-negative")
	}
	return nil
}

// -------------------- Auth --------------------

type AuthConfig struct {
	JWT   JWTConfig   `mapstructure:"jwt"`
	OTP   OTPConfig   `mapstructure:"otp"`
	OAuth OAuthConfig `mapstructure:"oauth"`
}

type JWTConfig struct {
	Algorithm             string        `mapstructure:"algorithm"`
	PublicKey             string        `mapstructure:"public_key"`
	PrivateKey            string        `mapstructure:"private_key"`
	AccessTokenExpiresIn  time.Duration `mapstructure:"access_token_expires_in"`
	RefreshTokenExpiresIn time.Duration `mapstructure:"refresh_token_expires_in"`
}

type OTPConfig struct {
	ExpiresIn time.Duration `mapstructure:"expires_in"`
	Secret    string        `mapstructure:"secret"`
	Length    int           `mapstructure:"length"`
}

type OAuthConfig struct {
	FrontendURL string        `mapstructure:"frontend_url"`
	Google      OAuthProvider `mapstructure:"google"`
	Facebook    OAuthProvider `mapstructure:"facebook"`
	Apple       OAuthProvider `mapstructure:"apple"`
}

type OAuthProvider struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// -------------------- Loading --------------------

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func NewConfig() (*Config, error) {
	return LoadConfig("config/config.yaml")
}
