# stealth

Stealth is a go interface to write/read from secret stores.

The current storage implementation uses [AWS System Manger Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html). Previously, it used a fork of [unicreds](https://github.com/Versent/unicreds).

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

Stealth works with the IdentityEngineer SSO Role/Profile to write to the operations or operations-dev account (depending on the --environment value).
```bash
    ./stealth write --assume --environment [production OR development] -- service [service-name] --key [key name] --value [key value]
```

If you're using the --assume flag and you are encountering permission issues, try the following before running stealth again:

```bash
    export AWS_PROFILE=[IdentityEngineer Profile Name]
```

# tests

To run tests, use:

```bash
    make test
```

This creates, updates, and reads secrets from the ci-test environment secret store, using the AWS credentials in your local environment.

# license

[Apache 2.0](./LICENSE)

# usage at Clever

Stealth is owned by #eng-security. For more info, see http://go/stealth.
