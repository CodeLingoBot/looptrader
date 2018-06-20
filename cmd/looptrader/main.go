package main

import (
    "os"
    "os/signal"
    "syscall"

	"github.com/morya/looptrader/conf"
	"github.com/morya/looptrader/log"
	"github.com/morya/looptrader/stratage"
)


func join(stra *stratage.Stratage) {
    s := make(chan os.Signal)
    signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
    <-s
    stra.stop()
}


func main() {
	conf.Init()
	log.Init()

	stra, err := stratage.NewStratage(conf.GetConfiguration())
	if err != nil {
		log.Logger.Panicf("create stratage failed. %s\n", err)
	}

	stra.run()
    join(stra)
}
