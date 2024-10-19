//go:build !windows && !plan9 && !nacl
// +build !windows,!plan9,!nacl

package syslog_test

import (
	"fmt"

	gosyslog "log/syslog"

	"github.com/a69/kit.go/log"
	"github.com/a69/kit.go/log/level"
	"github.com/a69/kit.go/log/syslog"
)

func ExampleNewSyslogLogger_defaultPrioritySelector() {
	// Normal syslog writer
	w, err := gosyslog.New(gosyslog.LOG_INFO, "experiment")
	if err != nil {
		fmt.Println(err)
		return
	}

	// syslog logger with logfmt formatting
	logger := syslog.NewSyslogLogger(w, log.NewLogfmtLogger)
	logger.Log("msg", "info because of default")
	logger.Log(level.Key(), level.DebugValue(), "msg", "debug because of explicit level")
}
