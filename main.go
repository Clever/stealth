package main

import (
	"bufio"
	"fmt"
	"github.com/Clever/stealth/store"
	"github.com/alecthomas/kingpin"
	"log"
	"os"
	"strings"
)

var (
	app         = kingpin.New("stealth", "The interface to Clever's secret store.")
	cmdDupes    = app.Command("dupes", "Finds duplicate values of a secret.")
	environment = cmdDupes.Flag("environment", "Environment that the secret belongs to.").Required().String()
	service     = cmdDupes.Flag("service", "Service that key belongs to.").Required().String()
	key         = cmdDupes.Flag("key", "Key to find duplicate values of.").Required().String()
	updateWith  = cmdDupes.Flag("update-with", "Value to update the duplicate values with.").Default("").String()
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch command {
	case cmdDupes.FullCommand():
		s := store.NewUnicredsStore()
		env := store.ProductionEnvironment
		if *environment == "development" {
			env = store.DevelopmentEnvironment
		} else if *environment != "production" {
			log.Fatal("Environment flag must be 'development' or 'production'")
		}
		id := store.SecretIdentifier{Environment: env, Service: *service, Key: *key}
		secret, err := s.Read(id)
		if err != nil {
			log.Fatal(err)
		}

		envs := [2]store.Environment{store.DevelopmentEnvironment, store.ProductionEnvironment}
		if *updateWith == "" {
			fmt.Println("Matching secret IDs")
			fmt.Println("===================")
		}
		for _, e := range envs {
			ids, err := s.ListAll(e)
			if err != nil {
				log.Fatal(err)
			}
			for _, id := range ids {
				newSecret, err := s.Read(id)
				if err != nil {
					log.Fatal(err)
				}
				if newSecret.Data == secret.Data {
					if *updateWith != "" {
						askForConfirmation("Are you sure you want to update the secret " + id.String() + "?")
						_, err := s.Update(id, *updateWith)
						if err != nil {
							log.Fatal(err)
							return
						}
					} else {
						fmt.Println(id.EnvironmentString() + "." + id.Service + "." + id.Key)
					}
				}
			}
		}
	}
}

// askForConfirmation asks the user for confirmation.
// See https://gist.github.com/m4ng0squ4sh/3dcbb0c8f6cfe9c66ab8008f55f8f28b
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
