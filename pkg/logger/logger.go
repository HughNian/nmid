package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger       *log.Logger
	nLogger      *zap.SugaredLogger
	stderrLogger *zap.SugaredLogger
	fileLogger   *zap.SugaredLogger
)

const (
	TIME_FORMAT = "20060102"
)

func init() {
	godotenv.Load("./.env")
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
	var logFileConstName, logFileName string
	if logConfig == nil {
		logDir = os.Getenv("LOG_DIR")
		logFileConstName = os.Getenv("LOGFILENAME")
	} else {
		logDir = logConfig.LogDir
		logFileConstName = logConfig.StdoutFilename
	}
	logFileName = fmt.Sprintf("%s_%s", time.Now().Format(TIME_FORMAT), logFileConstName)

	encoderConfig := defaultConfig()

	makeLogEnv(logDir, logFileName)
	logger, err := newLogFile(filepath.Join(logDir, logFileName), logMaxCacheCount)
	if err != nil {
		log.Fatalln("log file err", err.Error())
		os.Exit(1)
	}

	opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(1)}

	stderrSyncer := zapcore.AddSync(os.Stderr)
	stderrCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), stderrSyncer, level)
	stderrLogger = zap.New(stderrCore, opts...).Sugar()

	gatewaySyncer := zapcore.AddSync(logger)
	gatewayCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), gatewaySyncer, level)
	fileLogger = zap.New(gatewayCore, opts...).Sugar()

	defaultCore := zapcore.NewTee(gatewayCore, stderrCore)
	nLogger = zap.New(defaultCore, opts...).Sugar()

	//建立不同日期日志
	go func() {
		for {
			last := time.Now().Format(TIME_FORMAT)
			time.Sleep(1 * time.Second)
			now := time.Now().Format(TIME_FORMAT)

			if last != now {
				logFileName = fmt.Sprintf("%s_%s", time.Now().Format(TIME_FORMAT), logFileConstName)
				makeLogEnv(logDir, logFileName)
				logger, err = newLogFile(filepath.Join(logDir, logFileName), logMaxCacheCount)
				if err != nil {
					log.Fatalln("log file err", err.Error())
					os.Exit(1)
				}

				opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(1)}

				stderrSyncer := zapcore.AddSync(os.Stderr)
				stderrCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), stderrSyncer, level)
				stderrLogger = zap.New(stderrCore, opts...).Sugar()

				gatewaySyncer := zapcore.AddSync(logger)
				gatewayCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), gatewaySyncer, level)
				fileLogger = zap.New(gatewayCore, opts...).Sugar()

				defaultCore := zapcore.NewTee(gatewayCore, stderrCore)
				nLogger = zap.New(defaultCore, opts...).Sugar()
			}
		}
	}()
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
