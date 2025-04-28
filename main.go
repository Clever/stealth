package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Clever/stealth/store"
	"github.com/Clever/stealth/store/util"
	"github.com/alecthomas/kingpin"
)

var (
	app = kingpin.New("stealth", "The interface to Clever's secret store.")

	cmdDupes        = app.Command("dupes", "Finds duplicate values of a secret.")
	dupeEnvironment = cmdDupes.Flag("environment", "Environment that the secret belongs to.").Required().String()
	dupeService     = cmdDupes.Flag("service", "Service that key belongs to.").Required().String()
	dupeKey         = cmdDupes.Flag("key", "Key to find duplicate values of.").Required().String()
	updateWith      = cmdDupes.Flag("update-with", "Value to update the duplicate values with.").Default("").String()

	cmdDelete         = app.Command("delete", "Deletes all versions of a secret.")
	deleteEnvironment = cmdDelete.Flag("environment", "Environment that the secret belongs to.").Required().String()
	deleteService     = cmdDelete.Flag("service", "Service that key belongs to.").Required().String()
	deleteKey         = cmdDelete.Flag("key", "Key to find duplicate values of.").Required().String()

	cmdWrite         = app.Command("write", "Write a new version of a secret.")
	writeEnvironment = cmdWrite.Flag("environment", "Environment that the secret belongs to.").Required().String()
	writeService     = cmdWrite.Flag("service", "Service that the key belongs to.").Required().String()
	writeKey         = cmdWrite.Flag("key", "Key to write.").Required().String()
	writeValue       = cmdWrite.Flag("value", "Value to write.").Required().String()

	cmdHealth         = app.Command("health", "Checks for health of all secrets for a service across 4 AWS regions, ensuring there is no discrepancies in values.")
	healthEnvironment = cmdHealth.Flag("environment", "Environment that the secret belongs to.").Required().String()
	healthService     = cmdHealth.Flag("service", "Service that the key belongs to.").Required().String()

	assumeRole = app.Flag("assume", "If set, stealth will assume the SecretsManagement role (based on --environment)").Bool()
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch command {
	case cmdDupes.FullCommand():
		s := store.NewParameterStore(50, *dupeEnvironment, *assumeRole)
		id := store.SecretIdentifier{Environment: getEnvironment(*dupeEnvironment), Service: *dupeService, Key: *dupeKey}
		envs := []store.Environment{store.DevelopmentEnvironment, store.ProductionEnvironment}

		dupes, err := util.FindDupes(s, id, envs)
		if err != nil {
			log.Fatal(err)
		}
		if *updateWith == "" {
			fmt.Println("Matching secret IDs")
			fmt.Println("===================")
			for _, dupe := range dupes {
				fmt.Println(dupe.String())
			}
		} else {
			for _, dupe := range dupes {
				if askForConfirmation("Are you sure you want to update the secret " + dupe.String() + "?") {
					_, err := s.Update(dupe, *updateWith)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	case cmdDelete.FullCommand():
		s := store.NewParameterStore(50, *deleteEnvironment, *assumeRole)
		id := store.SecretIdentifier{Environment: getEnvironment(*deleteEnvironment), Service: *deleteService, Key: *deleteKey}
		if askForConfirmation("Are you sure you want to delete the secret " + id.String() + "?") {
			s.Delete(id)
		}

	case cmdWrite.FullCommand():
		s := store.NewParameterStore(50, *writeEnvironment, *assumeRole)
		id := store.SecretIdentifier{Environment: getEnvironment(*writeEnvironment), Service: *writeService, Key: *writeKey}
		// TODO: allow value to be a pointer to a file, or stdin
		if err := createOrUpdate(s, id, *writeValue); err != nil {
			log.Fatalf("Failed to write secret: %s", err)
		}
		fmt.Printf("Wrote secret %s\n", id.String())

	case cmdHealth.FullCommand():
		s := store.NewParameterStore(50, *healthEnvironment, *assumeRole)
		var stateOfSecrets = map[string]string{}
		for _, region := range s.GetOrderedRegions() {
			s.ParamRegion = region
			fmt.Printf("Checking store region %s\n", s.ParamRegion)
			var secrets []store.SecretIdentifier
			var err error
			var secretValue store.Secret
			if secrets, err = s.List(getEnvironment(*healthEnvironment), *healthService); err != nil {
				log.Fatalf("Failed to list secrets for : %s in %s: %s", *healthService, *healthEnvironment, err)
			}
			for _, id := range secrets {
				if secretValue, err = s.Read(id); err != nil {
					fmt.Printf("Error reading secret %s in region %s. %s \n", id.String(), region, err)
				}
				if val, ok := stateOfSecrets[id.String()]; ok {
					if secretValue.Data != val {
						fmt.Printf("Secret %s differs in region %s from us-west-1. \n", id.String(), region)
					}
				} else {
					stateOfSecrets[id.String()] = secretValue.Data
				}
			}
			fmt.Printf("Finished checking secrets in region %s.\n", region)
		}
	}

}

// getEnvironment returns the Environment enum value based on the string, or fatally errors if the string
// is not 'development' or 'production'
func getEnvironment(environment string) store.Environment {
	if environment == "development" {
		return store.DevelopmentEnvironment
	} else if environment == "ci-test" {
		return store.CITestEnvironment
	} else if environment != "production" {
		log.Fatal("Environment flag must be 'development' or 'production'")
	}
	return store.ProductionEnvironment
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

func createOrUpdate(s store.SecretStore, id store.SecretIdentifier, value string) error {
	err := s.Create(id, value)
	if err != nil {
		if _, ok := err.(*store.IdentifierAlreadyExistsError); !ok {
			return err
		}
	}

	_, err = s.Update(id, value)
	return err
}
