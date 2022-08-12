# gov-okta-addon

`gov-okta-addon` is an addon to integrate Okta with Governor.

## Usage

Updates to Okta are triggered both by a reconciliation loop as well as change events from governor.  During time based
reconciliation, `gov-okta-addon` requests all of the groups from `governor` and ensures those groups exist in Okta and are
configured with the same Github organizations.  Group membership is also reconciled, ensuring that all group members of
managed groups in `governor` are also members of the group in Okta (TODO).

`gov-okta-addon` also subscribes to the `governor` event stream where change events are published.  The events published
by `governor` contain the group and/or user id that changed and the type of action.  Events are publish on NATS subjects
dedicated to the resource type ie. `equinixmetal.governor.events.groups` for group events.  When `gov-okta-addon` receives
an event, it reacts by requesting information from governor about the included resource IDs and making the required changes in Okta.

## Development

`gov-okta-addon` includes a `docker-compose.yml` and a `Makefile` to make getting started easy.

`make docker-up` will start a basic NATS server and `gov-okta-addon`.
