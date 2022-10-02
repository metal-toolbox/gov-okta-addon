# gov-okta-addon

`gov-okta-addon` is an addon to integrate Okta with Governor.

## Usage

Updates to Okta are triggered both by a reconciliation loop as well as change events from Governor.  During time based
reconciliation, `gov-okta-addon` requests all of the groups from Governor and ensures those groups exist in Okta and are
configured with the same Github organizations.  Group membership is also reconciled, ensuring that all group members of
managed groups in Governor are also members of the group in Okta.

`gov-okta-addon` subscribes to the Governor event stream where change events are published.  The events published
by Governor contain the group and/or user id that changed and the type of action.  Events are publish on NATS subjects
dedicated to the resource type ie. `equinixmetal.governor.events.groups` for group events.  When `gov-okta-addon` receives
an event, it reacts by requesting information from Governor about the included resource IDs and making the required
changes in Okta.

## Syncing to governor

`gov-okta-addon` ships with a sync command to sync resources from Okta into `governor`. It has a `--dry-run` flag which
is helpful to see what resources would be affected.

### Sync users

`gov-okta-addon sync users` will sync users from Okta to governor based on the `id` in their Okta profile
and their `external_id` in Governor.

### Sync groups

`gov-okta-addon sync groups` will sync groups from Okta to governor based on the group slug and the `governor_id`
in their Okta profile. Groups that exist in Okta but not in governor will be created, and groups that exist in
governor but not in Okta will be deleted. Optionally, you can specify `--skip-okta-update` to avoid making changes
to the Okta group (i.e. setting the `governor_id`), and `--selector-prefix` to only sync specific groups.

This command will also associate any organizations with the group based on the assigned applications in Okta, but
it will not sync the members of the group.

### Sync group members

`gov-okta-addon sync members` will sync group members from Okta to governor. Group members that exist in Okta but not
in governor will be added to the governor group, and governor group members that do not exist in the Okta group will
be removed from the group. The groups and users must already exist in governor or they will be skipped.

## Development

`gov-okta-addon` includes a `docker-compose.yml` and a `Makefile` to make getting started easy.

`make docker-up` will start a basic NATS server and `gov-okta-addon`.

### Pre-requisites for running locally

You can run `gov-okta-addon` against your local Governor instance (if you don't already have one, follow the directions [here](https://github.com/equinixmetal/governor/blob/main/README.md#running-governor-locally) and also set up a [local Hydra](https://github.com/equinixmetal/governor/blob/main/README.md#governor-api)).

The first time you'll need to create a local hydra client for `gov-okta-addon-governor`, after that you can just export the secret:

```
export GOA_GOVERNOR_CLIENT_SECRET="$(echo $$RANDOM|md5)"
```

```
docker-compose exec hydra hydra clients create \
    --endpoint http://hydra:4445/ \
    --audience http://api:3001/ \
    --id gov-okta-addon-governor \
    --secret ${GOA_GOVERNOR_CLIENT_SECRET} \
    --grant-types client_credentials \
    --response-types token,code \
    --token-endpoint-auth-method client_secret_post \
    --scope  write,read:governor:users,read:governor:groups,read:governor:organizations
```

Export the required env variables to point to our local Governor, NATS and Hydra, and the Sandbox Okta instance:

```
export GOA_NATS_URL="nats://127.0.0.1:4222"
export GOA_NATS_TOKEN="notused"
export GOA_OKTA_NOCACHE=true
export GOA_OKTA_URL="https://equinixmetal.oktapreview.com"
export GOA_GOVERNOR_URL="http://127.0.0.1:3001"
export GOA_GOVERNOR_AUDIENCE="http://api:3001/"
export GOA_GOVERNOR_TOKEN_URL="http://127.0.0.1:4444/oauth2/token"
export GOA_GOVERNOR_CLIENT_ID="gov-okta-addon-governor"
```

Also ensure you have the following secrets exported:

```
export GOA_OKTA_TOKEN="REPLACE"
export GOA_GOVERNOR_CLIENT_SECRET="REPLACE"
```

### Testing addon serve locally

Create a local audit log for testing in the `gov-okta-addon` directory:

```
touch audit.log
```

Start the addon (adjust the flags as needed):

```
go run . serve --audit-log-path=audit.log --pretty --development --debug --dry-run
```

### Testing addon sync locally

Run the user sync (adjust the flags as needed):

```
go run . sync users --pretty --development --debug --dry-run
```

You can run the groups and members sync in the same way.
