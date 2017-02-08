package main

import (
	"fmt"
	"log"
	"minifileserver/sockhandler"
	"os"
	"path/filepath"
	"time"

	. "github.com/ecofast/iniutils"
	. "github.com/ecofast/sysutils"
)

const (
	listenPort            = 7000
	reportConnNumInterval = 10 * 60
)

func main() {
	startService()
}

func startService() {
	port := listenPort
	filePath := GetApplicationPath()
	tickerInterval := reportConnNumInterval
	ininame := ChangeFileExt(os.Args[0], ".ini")
	if FileExists(ininame) {
		port = IniReadInt(ininame, "setup", "port", port)
		filePath = IncludeTrailingBackslash(IniReadString(ininame, "setup", "filepath", filePath))
		tickerInterval = IniReadInt(ininame, "setup", "reportinterval", tickerInterval)
	} else {
		IniWriteInt(ininame, "setup", "port", port)
		IniWriteString(ininame, "setup", "filepath", filePath)
		IniWriteInt(ininame, "setup", "reportinterval", tickerInterval)
	}
	filePath = filepath.ToSlash(filePath)
	fmt.Printf("监听端口：%d\n", port)
	fmt.Printf("文件目录：%s\n", filePath)
	fmt.Printf("连接数报告间隔：%d\n", tickerInterval)

	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Second)
	go func() {
		for range ticker.C {
			log.Printf("[Ticker] 当前连接数：%d\n", sockhandler.Conns.Count())
		}
	}()

	sockhandler.Run(port, filePath)
}
