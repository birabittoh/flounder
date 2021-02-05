package main

import (
	"flag"
	"fmt"
	// "github.com/go-co-op/gocron"
	"github.com/gorilla/sessions"
	"io"
	"log"
	"os"
	"sync"
	// "time"
)

var c Config // global var to hold static configuration

func main() {
	configPath := flag.String("c", "flounder.toml", "path to config file") // doesnt work atm
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("expected 'admin' or 'serve' subcommand")
		os.Exit(1)
	}

	var err error
	c, err = getConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	logFile, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	initializeDB()

	cookie := generateCookieKeyIfDNE()
	SessionStore = sessions.NewCookieStore(cookie)
	// load domains in memory
	refreshDomainMap()

	// Background workers
	if c.AnalyticsDBFile != "" {
		go dumpLogsWorker()
	}

	switch args[0] {
	case "serve":
		// s1.StartAsync()
		wg := new(sync.WaitGroup)
		wg.Add(2)
		go func() {
			runHTTPServer()
			wg.Done()
		}()
		go func() {
			runGeminiServer()
			wg.Done()
		}()
		wg.Wait()
	case "admin":
		runAdminCommand()
	case "dumplogs":
		dumpLogs()
	}
}
