package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RasmusLindroth/indy/pkg/config"
	"github.com/RasmusLindroth/indy/pkg/fetcher"

	"github.com/RasmusLindroth/indy/pkg/database"
	"github.com/RasmusLindroth/indy/pkg/web"

	"github.com/RasmusLindroth/indy/pkg/news"
)

//App holds shared objects
type App struct {
	Sites []*news.Site
	DB    *database.Handler
	Web   *web.Handler
}

func (app *App) cron() {
	for {
		go fetcher.FetchSites(app.Sites, app.DB)
		<-time.After(15 * time.Minute)
	}
}

func main() {
	var confPath string
	flag.StringVar(&confPath, "conf", "", "path to your config.yml")
	flag.Parse()

	if confPath == "" {
		fmt.Fprintf(os.Stderr, "You need to set -conf /path/to/config.yml\n")
		os.Exit(1)
	}

	conf, err := config.ParseFile(confPath)

	if err != nil {
		log.Fatalln(err)
	}
	app := &App{}

	app.Sites = conf.Sites

	dbHandler, err := database.New(conf.Database.Username, conf.Database.Password, conf.Database.Port)
	if err != nil {
		log.Fatalf("Couldn't connect to db: %v\n", err)
	}
	defer dbHandler.Close()

	app.DB = dbHandler

	go app.cron()

	app.Web = web.New(app.DB, app.Sites, conf.Web.Files)
	app.Web.StartServer(conf.Web.Port)
}
