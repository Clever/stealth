package main

import (
	"fmt"
	"github.com/Clever/stealth/store"
	"github.com/alecthomas/kingpin"
	"os"
)

var (
	app          = kingpin.New("stealth", "The interface to Clever's secret store.")
	cmdFindDupes = app.Command("find-dupes", "Finds duplicate values of a secret.")
	environment  = cmdFindDupes.Flag("environment", "Environment that the secret belongs to.").Required().String()
	service      = cmdFindDupes.Flag("service", "Service that key belongs to.").Required().String()
	key          = cmdFindDupes.Flag("key", "Key to find duplicate values of.").Required().String()
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch command {
	case cmdFindDupes.FullCommand():
		s := store.NewUnicredsStore()
		env := store.ProductionEnvironment
		if *environment == "development" {
			env = store.DevelopmentEnvironment
		} else if *environment != "production" {
			fmt.Println("Environment flag must be 'development' or 'production'")
			return
		}
		id := store.SecretIdentifier{Environment: env, Service: *service, Key: *key}
		secret, err := s.Read(id)
		if err != nil {
			fmt.Println(err)
			return
		}

		matchingIds := make([]store.SecretIdentifier, 0)

		envs := [2]store.Environment{store.DevelopmentEnvironment, store.ProductionEnvironment}

		for _, e := range envs {
			ids, err := s.ListAll(e)
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, id := range ids {
				newSecret, err := s.Read(id)
				if err != nil {
					fmt.Println(err)
					return
				}
				if newSecret.Data == secret.Data {
					matchingIds = append(matchingIds, id)
				}
			}
		}
		fmt.Println("Matching IDs")
		fmt.Println("============")
		for _, id = range matchingIds {
			fmt.Println(id.EnvironmentString() + "." + id.Service + "." + id.Key)
		}
	}
}
