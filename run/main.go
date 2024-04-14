package main

import (
	"context"
	"log"
	"os"
	"reflect"
	"runtime"
	mc_vultr_gov "sebekerga/vultr_minecraft_governor"
	routines "sebekerga/vultr_minecraft_governor/routines"
	routines_create "sebekerga/vultr_minecraft_governor/routines/createserver"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
	tele "gopkg.in/telebot.v3"
)

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	tgToken := os.Getenv(mc_vultr_gov.TELEGRAM_TOKEN_KEY)
	botId := strings.Split(tgToken, ":")[0]
	obfuscatedToken := strings.Repeat("*", len(tgToken)-len(botId)-1)

	log.Printf("Token: %s:%s", botId, obfuscatedToken)

	pref := tele.Settings{
		Token:  tgToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	apiKey := os.Getenv(mc_vultr_gov.VULTR_API_KEY_KEY)
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	b.Handle("/create", func(c tele.Context) error {
		printHandler := func(level routines.PrintLevel, message string) {
			switch level {
			case routines.INFO:
				c.Reply(message)
			case routines.ERROR:
				c.Reply(message)
			}
		}

		creationCtx := routines_create.InitContext(ctx, vultrClient)
		routine := routines.InitRoutine[routines_create.Ctx](routines_create.CreatingServerEntry, creationCtx, printHandler)

		for !routine.Finished() {
			log.Print("Running routine step", GetFunctionName(routine.QueuedFunction))
			err := routine.Step()
			if err != nil {
				return err
			}
		}

		return nil
	})

	b.Start()
}
