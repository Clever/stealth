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
	app               = kingpin.New("stealth", "The interface to Clever's secret store.")
	cmdDupes          = app.Command("dupes", "Finds duplicate values of a secret.")
	dupeEnvironment   = cmdDupes.Flag("environment", "Environment that the secret belongs to.").Required().String()
	dupeService       = cmdDupes.Flag("service", "Service that key belongs to.").Required().String()
	dupeKey           = cmdDupes.Flag("key", "Key to find duplicate values of.").Required().String()
	updateWith        = cmdDupes.Flag("update-with", "Value to update the duplicate values with.").Default("").String()
	cmdDelete         = app.Command("delete", "Deletes all versions of a secret.")
	deleteEnvironment = cmdDelete.Flag("environment", "Environment that the secret belongs to.").Required().String()
	deleteService     = cmdDelete.Flag("service", "Service that key belongs to.").Required().String()
	deleteKey         = cmdDelete.Flag("key", "Key to find duplicate values of.").Required().String()
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch command {
	case cmdDupes.FullCommand():
		findDupes()
	case cmdDelete.FullCommand():
		deleteSecret()
	}
}

func findDupes() {
	s := store.NewUnicredsStore()
	env := store.ProductionEnvironment
	if *dupeEnvironment == "development" {
		env = store.DevelopmentEnvironment
	} else if *dupeEnvironment != "production" {
		log.Fatal("Environment flag must be 'development' or 'production'")
	}
	id := store.SecretIdentifier{Environment: env, Service: *dupeService, Key: *dupeKey}
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
					if askForConfirmation("Are you sure you want to update the secret " + id.String() + "?") {
						_, err := s.Update(id, *updateWith)
						if err != nil {
							log.Fatal(err)
							return
						}
					}
				} else {
					fmt.Println(id.EnvironmentString() + "." + id.Service + "." + id.Key)
				}
			}
		}
	}
}

func deleteSecret() {
	s := store.NewUnicredsStore()
	env := store.ProductionEnvironment
	if *deleteEnvironment == "development" {
		env = store.DevelopmentEnvironment
	} else if *deleteEnvironment != "production" {
		log.Fatal("Environment flag must be 'development' or 'production'")
	}
	id := store.SecretIdentifier{Environment: env, Service: *deleteService, Key: *deleteKey}
	if askForConfirmation("Are you sure you want to update the secret " + id.String() + "?") {
		s.Delete(id)
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
