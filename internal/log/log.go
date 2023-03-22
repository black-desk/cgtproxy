package log

import (
	"log"
	"log/syslog"
	"sync"
)

var (
	debugLog   *log.Logger
	infoLog    *log.Logger
	noticeLog  *log.Logger
	warningLog *log.Logger
	errLog     *log.Logger
	critLog    *log.Logger
	alertLog   *log.Logger
	emergLog   *log.Logger
)

var once sync.Once

func init() {
	priorities := map[**log.Logger]syslog.Priority{
		&debugLog:   syslog.LOG_DEBUG,
		&infoLog:    syslog.LOG_INFO,
		&noticeLog:  syslog.LOG_NOTICE,
		&warningLog: syslog.LOG_WARNING,
		&errLog:     syslog.LOG_ERR,
		&critLog:    syslog.LOG_CRIT,
		&alertLog:   syslog.LOG_ALERT,
		&emergLog:   syslog.LOG_EMERG,
	}

	for logPtr := range priorities {
		var err error
		*logPtr, err = syslog.NewLogger(syslog.LOG_USER|priorities[logPtr], log.Llongfile)
		if err != nil {
			panic(err)
		}
	}
}

func Debug() *log.Logger {
	return debugLog
}

func Info() *log.Logger {
	return infoLog
}

func Notice() *log.Logger {
	return noticeLog
}

func Warning() *log.Logger {
	return warningLog
}

func Err() *log.Logger {
	return errLog
}

func Crit() *log.Logger {
	return critLog
}

func Alert() *log.Logger {
	return alertLog
}

func Emerg() *log.Logger {
	return emergLog
}
