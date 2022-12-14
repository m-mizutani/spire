package utils

import "github.com/m-mizutani/zlog"

var logger = zlog.New()

func Logger() *zlog.Logger {
	return logger
}

func RenewLogger(options ...zlog.Option) {
	logger = logger.Clone(options...)
}
