# stealth

Stealth is a go interface to write/read from secret stores.

The current storage implementation uses [AWS System Manger Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html). Previously, it used our fork of [unicreds](https://github.com/Clever/unicreds), which is a go port of [credstash](https://github.com/fugue/credstash), which uses AWS [DynamoDB](https://aws.amazon.com/dynamodb/) and [KMS](https://aws.amazon.com/kms/).

# usage

Stealth can be run standalone for certain administrative tasks. First you'll need to compile the binary:

```bash
    make build
```

To find all secrets that have the same value as an existing secret (for instance, to revoke a leaked secret):

```bash
    ./stealth dupes --environment [production OR development] --service [service-name] --key [key name]
```

You can replace all these values using this command:

```bash
    ./stealth dupes --environment [production OR development] --service [service-name] --key [key name] --update-with [value to replace with]
```

To delete a secret:

```bash
    ./stealth delete --environment [production OR development] --service [service-name] --key [key name]
```

To write a secret:

```bash
    ./stealth write --environment [production OR development] -- service [service-name] --key [key name] --value [key value]
```

To identify discrepancies in secret values across 4 U.S. regions of AWS.

```bash
    ./stealth health --environment=ENVIRONMENT --service=SERVICE
```

# tests

To run tests, use:

```bash
    make test
```

This creates, updates, and reads secrets from the ci-test environment secret store, using the AWS credentials in your local environment.

# setting up backend infrastructure

If you are using Terraform, you can use the module [tf-credstash](https://github.com/dfuentes/tf-credstash) to set up the necessary DynamoDB and KMS key for stealth. For example, to create a dev backend, you can use this terraform code:

```HCL
provider "aws" {}

module "stealth-dev" {
  source = "github.com/dfuentes/tf-credstash"
  key_alias = "alias/stealth-key-dev"
  table_name = "stealth-dev"
}
```

# license

[Apache 2.0](./LICENSE)

# usage at Clever

Stealth is co-owned by #eng-infra and #eng-security. For more info, see http://go/stealth
