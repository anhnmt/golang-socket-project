package config

import (
	"context"

	"go.uber.org/zap"
)

type Config struct {
	Ctx context.Context
	Log *zap.Logger
}

func NewConfig(log *zap.Logger) *Config {
	return &Config{
		Ctx: context.Background(),
		Log: log,
	}
}
