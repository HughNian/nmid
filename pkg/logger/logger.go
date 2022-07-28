package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"nmid-v2/pkg/model"
	"nmid-v2/pkg/utils"
	"os"
	"path/filepath"
)

var (
	nLogger      *zap.SugaredLogger
	stderrLogger *zap.SugaredLogger
	fileLogger   *zap.SugaredLogger
)

const (
	defaultLogDir      = "./log_dir"
	defaultLogFileName = "log_file.log"
)

func init() {
	NewLogger(nil)
}

func NewLogger(logConfig *model.LogConfig) {
	var level zapcore.Level
	level = zap.InfoLevel
	if logConfig != nil {
		if logConfig.Debug {
			level = zap.DebugLevel
		}
	}

	var logDir string
	var logFileName string
	if logConfig == nil {
		logDir = defaultLogDir
		logFileName = defaultLogFileName
	} else {
		logDir = logConfig.LogDir
		logFileName = logConfig.StdoutFilename
	}

	encoderConfig := defaultConfig()

	makeLogEnv(logDir, logFileName)
	lfile, err := newLogFile(filepath.Join(logDir, logFileName), logMaxCacheCount)
	if err != nil {
		log.Fatalln("log file err", err.Error())
		os.Exit(1)
	}

	opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(1)}

	stderrSyncer := zapcore.AddSync(os.Stderr)
	stderrCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), stderrSyncer, level)
	stderrLogger = zap.New(stderrCore, opts...).Sugar()

	gatewaySyncer := zapcore.AddSync(lfile)
	gatewayCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), gatewaySyncer, level)
	fileLogger = zap.New(gatewayCore, opts...).Sugar()

	defaultCore := zapcore.NewTee(gatewayCore, stderrCore)
	nLogger = zap.New(defaultCore, opts...).Sugar()
}

func defaultConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "nmind-logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func makeLogEnv(logDir, logFileName string) {
	if !utils.PathExist(logDir) {
		utils.CreateFile(logDir + "/" + logFileName)
	}
}
