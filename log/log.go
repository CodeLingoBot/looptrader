package log

import (
	// "fmt"
	// "strings"
	"os"

	"github.com/morya/fcoinExchange/conf"
	"github.com/morya/utils/log"
)

var (
	Logger *log.Logger
)

func Init() {
	var c = conf.GetConfiguration()
	var o = os.Stdout
	// var err error

	//	if len(c.LogFile) != 0 {
	//		o, err = log.NewRollingFile(c.LogFile, log.DailyRolling)
	//		if err != nil {
	//			log.Info("create rolling file failed, using stdout")
	//			o = os.Stdout
	//		}
	//	}

	Logger = log.New(o, "")

	Logger.SetFlags(log.LstdFlags | log.Lshortfile)
	Logger.SetLevelString(c.LogLevel)
}
