package gosmpp

import (
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
