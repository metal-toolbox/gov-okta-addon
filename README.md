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

`gov-okta-addon` ships with a sync command to sync resources from Okta into `governor`.

### Sync users

`gov-okta-addon sync users` will sync users from Okta to governor based on the `id` in their Okta profile
and their `external_id` in Governor.

## Development

`gov-okta-addon` includes a `docker-compose.yml` and a `Makefile` to make getting started easy.

`make docker-up` will start a basic NATS server and `gov-okta-addon`.

### Testing user sync locally

You can try running the user sync against your local Governor instance (if you don't already have one, follow the directions [here](https://github.com/equinixmetal/governor/blob/main/README.md#running-governor-locally) and also set up a [local Hydra](https://github.com/equinixmetal/governor/blob/main/README.md#governor-api)).

You can point to the production Okta instance (the default), just set the token:
```
export GOA_OKTA_TOKEN="..."
```

Use the client-secret from your local Hydra and you should be able to run this successfully without making any changes:
```
go run . sync users --pretty --governor-url "http://localhost:3001" --governor-audience "http://api:3001/" --governor-token-url "http://localhost:4444/oauth2/token" --governor-client-id "governor" --governor-client-secret "SECRET_FROM_HYDRA" --dry-run

2022-09-09T12:52:33.876-0400	INFO	cmd/sync_users.go:42	starting sync to governor	{"app": "gov-okta-addon", "dry-run": true}
2022-09-09T12:52:34.002-0400	INFO	cmd/sync_users.go:167	starting to sync missing okta users into governor	{"app": "gov-okta-addon", "dry-run": true}
2022-09-09T12:52:34.002-0400	INFO	okta/users.go:40	listing users with func	{"app": "gov-okta-addon"}
<snip>
2022-09-09T12:52:36.561-0400	INFO	cmd/sync_users.go:193	starting to clean orphan governor users	{"app": "gov-okta-addon", "dry-run": true}
2022-09-09T12:52:36.564-0400	INFO	cmd/sync_users.go:179	completed user sync	{"app": "gov-okta-addon", "created": 365, "deleted": 0, "skipped": 0}
```

You can remove the `--dry-run` flag if you want to sync the changes to governor.
