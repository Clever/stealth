# stealth

Interface to write/read from secret stores

Owned by eng-infra

## Deploying

```
ark start stealth -e production
```

## Design Qs:

- Is it OK to unintentially overwrite a key or should we have an update operation?
- Do we need to be able to read versioned config?
- What should be our scheme for key names?
    - how to make this easy to use for most eng making ark deploys, but also flexible to store other secrets
        - ark: `/deployment/catapult/<app>/<env>/` - secrets for catapult deployed apps
        - other: one example `/aws/rds/<env>` - AWS keys generated for RDS instance. not associated with any app yet. perhaps later associated with many apps.
    - should keys be stored based on app meaning or secret meaning?
        - how to model that multiple apps should both have access to same signalFX token?
        - if possible, we should token in 1 place for easy revocation, updates
            - is it always safe to revoke in one place -- we have to be thoughtful about all consumers in that case
