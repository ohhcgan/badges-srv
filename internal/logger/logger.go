package logger

import (
	"fmt"

	"github-badges-backend/internal/config"

	"go.uber.org/zap"
)

func Logger(env config.EnvType) (*zap.Logger, error) {
	switch env {
	case config.Production:
		return zap.NewProduction()
	case config.Development:
		return zap.NewDevelopment()
	}
	return nil, fmt.Errorf("Invalid environment %v", env)
}
