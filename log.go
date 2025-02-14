package gosmpp

import (
	"context"
	"github.com/gogf/gf/v2/os/glog"
)

var l *glog.Logger

func SetLog(inputLog *glog.Logger) {
	l = inputLog
}

func GetLog() *glog.Logger {
	if l == nil {
		panic("cursom log is nil")
	}
	return l
}

func GetLogIns() *glog.Logger {
	if l == nil {
		return nil
	}
	return l
}
func GInfof(ctx context.Context, format string, v ...interface{}) {
	glogIns := GetLogIns()
	if glogIns == nil {
		return
	}
	glogIns.Infof(ctx, format, v...)
}
