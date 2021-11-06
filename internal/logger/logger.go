package logger

import (
	"log"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger is a wrapper for zap.Logger
func NewLogger() *zap.Logger {
	cfg := zap.Config{
		Encoding:    "console",                           // encode kiểu json hoặc console
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel), // chọn InfoLevel có thể logger ở cả 3 level
		OutputPaths: []string{"stderr"},

		EncoderConfig: zapcore.EncoderConfig{
			// Cấu hình logging, sẽ không có stacktracekey
			MessageKey:   "message",
			TimeKey:      "time",
			LevelKey:     "level",
			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder, // Lấy dòng code bắt đầu logger
			EncodeLevel:  CustomLevelEncoder,         // Format cách hiển thị level logger
			EncodeTime:   SyslogTimeEncoder,          // Format hiển thị thời điểm logger
		},
	}

	logger, err := cfg.Build() // Build ra Logger
	// logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}

	return logger
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func Close(logger *zap.Logger) {
	defer func(logger *zap.Logger) {
		if err := logger.Sync(); err != nil {
			logger.Error("logger.Sync()",
				zap.Error(err),
			)
		}
	}(logger)
}
