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

`gov-okta-addon sync users` will sync users from Okta to governor based on the `pingSubject` in their Okta profile
and their `external_id` in Governor.

## Development

`gov-okta-addon` includes a `docker-compose.yml` and a `Makefile` to make getting started easy.

`make docker-up` will start a basic NATS server and `gov-okta-addon`.
