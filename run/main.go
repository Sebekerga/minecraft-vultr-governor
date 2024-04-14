package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	mc_vultr_gov "sebekerga/vultr_minecraft_governor"
	routines "sebekerga/vultr_minecraft_governor/routines"
	routines_create "sebekerga/vultr_minecraft_governor/routines/createserver"
	routines_remove "sebekerga/vultr_minecraft_governor/routines/removeserver"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
	tele "gopkg.in/telebot.v3"
)

const ROUTINE_LOG_SIZE = 10

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

	log.Println("Connecting to Vultr")
	apiKey := os.Getenv(mc_vultr_gov.VULTR_API_KEY_KEY)
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))
	log.Println("Connected to Vultr")

	printHandler := func(tgMessage *tele.Message, messageStack *[]string, level routines.PrintLevel, message string) {
		log.Printf("[%s] %s", level, strings.Replace(message, "\n", " ", -1))
		switch level {
		case routines.INFO:
			*messageStack = append(*messageStack, fmt.Sprintf("OK > %s", message))
		case routines.ERROR:
			*messageStack = append(*messageStack, fmt.Sprintf("ERROR > %s", message))
		}

		if len(*messageStack) > ROUTINE_LOG_SIZE {
			*messageStack = (*messageStack)[1:]
		}

		filledMessageStack := make([]string, ROUTINE_LOG_SIZE-len(*messageStack))
		for i := range filledMessageStack {
			filledMessageStack[i] = "> "
		}

		messageText := fmt.Sprintf("Starting server\n```routine_log\n%s\n```", strings.Join(append(*messageStack, filledMessageStack...), "\n"))
		b.Edit(tgMessage, messageText, &tele.SendOptions{
			ParseMode: tele.ModeMarkdownV2,
		})
	}

	b.Handle("/create", func(c tele.Context) error {
		botMessage, err := b.Reply(c.Message(), "Starting server")
		if err != nil {
			return err
		}

		messageStack := []string{}
		thisPrintHandler := func(level routines.PrintLevel, message string) {
			printHandler(botMessage, &messageStack, level, message)
		}

		creationCtx := routines_create.InitContext(ctx, vultrClient)
		routine := routines.InitRoutine[routines_create.Ctx](routines_create.CreatingServerEntry, creationCtx, thisPrintHandler)

		for !routine.Finished() {
			routineStepName := GetFunctionName(routine.QueuedFunction)
			routineStepNameSlice := strings.Split(routineStepName, "/")
			routineStepName = routineStepNameSlice[len(routineStepNameSlice)-1]

			log.Printf("Running routine step %s", routineStepName)
			err := routine.Step()
			if err != nil {
				return err
			}
		}

		return nil
	})

	b.Handle("/remove", func(c tele.Context) error {
		botMessage, err := b.Reply(c.Message(), "Stopping server")
		if err != nil {
			return err
		}

		messageStack := []string{}
		thisPrintHandler := func(level routines.PrintLevel, message string) {
			printHandler(botMessage, &messageStack, level, message)
		}
		removingCtx := routines_remove.InitContext(ctx, vultrClient)
		routine := routines.InitRoutine[routines_remove.Ctx](routines_remove.RemovingServerEntry, removingCtx, thisPrintHandler)

		for !routine.Finished() {
			routineStepName := GetFunctionName(routine.QueuedFunction)
			routineStepNameSlice := strings.Split(routineStepName, "/")
			routineStepName = routineStepNameSlice[len(routineStepNameSlice)-1]

			log.Printf("Running routine step %s", routineStepName)
			err := routine.Step()
			if err != nil {
				return err
			}
		}

		return nil
	})

	log.Println("Starting bot")
	b.Start()
}
