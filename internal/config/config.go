package config

import (
	"fmt"
	"os"
	"strconv"
)

type AppConfig struct {
	Name string
	Env  string
	Port string
}

type MySQLConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type WorkerConfig struct {
	Concurrency int
	Queue       string
}

type Config struct {
	App    AppConfig
	MySQL  MySQLConfig
	Redis  RedisConfig
	Worker WorkerConfig
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "TaskTracker"),
			Env:  getEnv("APP_ENV", "local"),
			Port: getEnv("APP_PORT", "8080"),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			Name:     getEnv("DB_NAME", "task_tracker"),
			User:     getEnv("DB_USER", "task_tracker_user"),
			Password: getEnv("DB_PASSWORD", "task_tracker_password"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Worker: WorkerConfig{
			Concurrency: getEnvAsInt("WORKER_CONCURRENCY", 10),
			Queue:       getEnv("WORKER_QUEUE", "default"),
		},
	}

	if cfg.MySQL.Host == "" || cfg.MySQL.Port == "" || cfg.MySQL.Name == "" || cfg.MySQL.User == "" {
		return Config{}, fmt.Errorf("invalid mysql config: required fields are empty")
	}

	return cfg, nil
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return intValue
}
