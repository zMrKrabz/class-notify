package main

import (
	"flag"
	"fmt"
	class_notify "github.com/zMrKrabz/class-notify"
	"github.com/zMrKrabz/class-notify/schools"
	"log"
	"os"
	"os/signal"
)

var (
	AUTH_TOKEN = ""
	GUILD_ID = ""
	MONGO_DB_URL = ""
	SCHOOL = ""
)

func main() {
	flag.StringVar(&AUTH_TOKEN,"auth", "", "discord authentication token")
	flag.StringVar(&GUILD_ID, "guild", "", "guild id if specified")
	flag.StringVar(&MONGO_DB_URL, "mongo", "mongodb://127.0.0.1:27017", "mongodb database url")
	flag.StringVar(&SCHOOL, "school", "", "school to connect to")
	flag.Parse()

	db := class_notify.Database{}
	if err := db.Connect(MONGO_DB_URL); err != nil {
		panic(fmt.Sprintf("error on connecting to mongodb database: %s", err))
	}

	var school schools.ISchool
	switch SCHOOL {
	case "GEORGIA_TECH":
		school = &schools.GeorgiaTech{}
	}

	bot := class_notify.Bot{
		DB: &db,
		School: school,
	}

	dg := class_notify.Discord{
		Bot: &bot,
	}
	if err := dg.Connect(AUTH_TOKEN, GUILD_ID); err != nil {
		panic(fmt.Sprintf("unable to ocnnect to discord: %s", err))
	}
	defer dg.Close()

	go bot.StartMonitor(dg.UpdateSubscriber)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press CTRL + C to exit")
	<- stop
	log.Println("gracefully shutting down")
}