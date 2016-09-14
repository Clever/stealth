# stealth

Stealth is a go interface to write/read from secret stores.

The current storage implementation uses our fork of [unicreds](https://github.com/Clever/unicreds), which is a go port of [credstash](https://github.com/fugue/credstash), which uses AWS [DynamoDB](https://aws.amazon.com/dynamodb/) and [KMS](https://aws.amazon.com/kms/).

Stealth is co-owned by #eng-infra and #security. For more info, see http://go/stealth

# tests

To run tests, use:

    make test

This creates, updates, and reads secrets from the secret store, using the AWS credentials in your local environment.
