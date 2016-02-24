package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia"
	"github.com/elos/gaia/agents"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"github.com/subosito/twilio"
	"golang.org/x/net/context"
)

const (
	TwilioAccountSid = "AC76d4c9975dfb641d9ae711c2f795c5a2"
	TwilioAuthToken  = "9ab82f10b0b6187d2c71589c46c96da6"
	TwilioFromNumber = "+16503810349"
)

func main() {
	var (
		addr   = flag.String("addr", "0.0.0.0", "address to listen on")
		port   = flag.Int("port", 80, "port to listen on")
		dbtype = flag.String("dbtype", "mongo", "type of database to use: (mem or mongo)")
		dbaddr = flag.String("dbaddr", "0.0.0.0", "address of database")
		appdir = flag.String("appdir", "app", "directory of maia build")
		db     data.DB
		err    error
	)

	flag.Parse()

	log.Printf("== Setting Up Database ==")
	log.Printf("\tDatabase Type: %s", *dbtype)
	switch *dbtype {
	case "mem":
		db = mem.NewDB()
	case "mongo":
		db, err = models.MongoDB(*dbaddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("\tConnected to mongo@%s", *dbaddr)
	default:
		log.Fatal("Unrecognized database type: '%s'", *dbtype)
	}
	log.Printf("== Set up Database ==")

	background := context.Background()

	log.Printf("== Connecting to Twilio ==")
	twilioClient := twilio.NewClient(TwilioAccountSid, TwilioAuthToken, nil)
	log.Printf("== Connected to Twilio ==")

	log.Printf("== Starting SMS Command Sessions ==")
	smsMux := services.NewSMSMux()
	go smsMux.Start(
		background,
		db,
		services.SMSFromTwilio(twilioClient, TwilioFromNumber),
	)
	log.Printf("== Started SMS Command Sessions ==")

	log.Printf("== Initiliazing Gaia Core ==")
	g := gaia.New(
		context.Background(),
		new(gaia.Middleware),
		&gaia.Services{
			AppFileSystem:      http.Dir(*appdir),
			SMSCommandSessions: smsMux,
			DB:                 db,
			Logger:             services.NewLogger(os.Stderr),
		},
	)
	log.Printf("== Initiliazed Gaia Core ==")

	log.Printf("== Starting Agents ===")
	user.Map(db, func(db data.DB, u *models.User) error {
		go agents.LocationAgent(background, db, u)
		go agents.TaskAgent(background, db, u)
		return nil
	})
	log.Printf("== Started Agents ===")

	log.Printf("== Starting HTTP Server ==")
	host := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("\tServing on %s", host)
	if err = http.ListenAndServe(host, g); err != nil {
		log.Fatal(err)
	}
	log.Printf("== Started HTTP Server ==")
}
