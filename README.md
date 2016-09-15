# stealth

Stealth is a go interface to write/read from secret stores.

The current storage implementation uses our fork of [unicreds](https://github.com/Clever/unicreds), which is a go port of [credstash](https://github.com/fugue/credstash), which uses AWS [DynamoDB](https://aws.amazon.com/dynamodb/) and [KMS](https://aws.amazon.com/kms/).

Stealth is co-owned by #eng-infra and #security. For more info, see http://go/stealth

# usage

Stealth can be run standalone for certain administrative tasks. First you'll need to compile the binary:

    make build

To find all secrets that have the same value as an existing secret (for instance, to revoke a leaked secret):

    ./stealth dupes --environment [production OR development] --service [service-name] --key [key name]

You can replace all these values using this command:

    ./stealth dupes --environment [production OR development] --service [service-name] --key [key name] --replace-with [value to replace with]

To delete a secret:

    ./stealth delete --environment [production OR development] --service [service-name] --key [key name]

# tests

To run tests, use:

    make test

This creates, updates, and reads secrets from the drone-test environment secret store, using the AWS credentials in your local environment.
