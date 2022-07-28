package logger

// Info is info level
func Info(args ...interface{}) {
	nLogger.Info(args...)
}

// Warn is warning level
func Warn(args ...interface{}) {
	nLogger.Warn(args...)
}

// Error is error level
func Error(args ...interface{}) {
	nLogger.Error(args...)
}

// Debug is debug level
func Debug(args ...interface{}) {
	nLogger.Debug(args...)
}

// Infof is format info level
func Infof(fmt string, args ...interface{}) {
	nLogger.Infof(fmt, args...)
}

// Warnf is format warning level
func Warnf(fmt string, args ...interface{}) {
	nLogger.Warnf(fmt, args...)
}

// Errorf is format error level
func Errorf(fmt string, args ...interface{}) {
	nLogger.Errorf(fmt, args...)
}

// Debugf is format debug level
func Debugf(fmt string, args ...interface{}) {
	nLogger.Debugf(fmt, args...)
}

// Fatal logs a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	nLogger.Fatal(args...)
}

// Fatalf logs a templated message, then calls os.Exit.
func Fatalf(fmt string, args ...interface{}) {
	nLogger.Fatalf(fmt, args...)
}
