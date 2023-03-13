# gov-okta-addon

`gov-okta-addon` is an addon to integrate Okta with Governor.

## Usage

Updates to Okta are triggered both by a reconciliation loop as well as change events from Governor.  During time based
reconciliation, `gov-okta-addon` requests all of the groups from Governor and ensures those groups exist in Okta and are
configured with the same Github organizations.  Group membership is also reconciled, ensuring that all group members of
managed groups in Governor are also members of the group in Okta.  Users are reconciled by making sure deleted users in
Governor and deleted in Okta (currently this is only logging), and the status of suspended/unsuspended users is updated
accordingly in Okta.

`gov-okta-addon` subscribes to the Governor event stream where change events are published.  The events published
by Governor contain the group and/or user id that changed and the type of action.  Events are published on NATS subjects
dedicated to the resource type ie. `equinixmetal.governor.events.groups` for group events.  When `gov-okta-addon` receives
an event, it reacts by requesting information from Governor about the included resource IDs and making the required
changes in Okta.

### Safe mode

There are two flags that can limit the changes that `gov-okta-addon` makes and just log `SKIP` messages instead.

`--skip-delete` is currently enabled by default and it will prevent the timed reconcile loop from deleting stuff in Okta (this
includes removing group members, removing application group assignments, or removing users). This flag does not apply to any of
the NATS events which will be processed normally.

`--dry-run` will prevent any changes from being made while the addon is running, including the reconcile loop and NATS events.

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
to the Okta group (i.e. setting the `governor_id`),  `--selector-prefix` to only sync specific groups, and
`--skip-groups "foo,bar,baz"` to skip syncing groups named `foo`, `bar` and `baz`.

This command will also associate any organizations with the group based on the assigned applications in Okta, but
it will not sync the members of the group.

### Sync group members

`gov-okta-addon sync members` will sync group members from Okta to governor. Group members that exist in Okta but not
in governor will be added to the governor group, and governor group members that do not exist in the Okta group will
be removed from the group. The groups and users must already exist in governor or they will be skipped.

## Development

`gov-okta-addon` includes a `docker-compose.yml` and a `Makefile` to make getting started easy.

`make docker-up` will start a basic NATS server and `gov-okta-addon`.

### Prereq to running locally with governor-api devcontainer

Follow the directions [here](https://github.com/equinixmetal/governor/blob/main/README.md#running-governor-locally) for starting the governor-api devcontainer.

The **first time** you'll need to create a local hydra client for `gov-okta-addon-governor` and copy the nats creds file. After that you can just export the env variables.

#### NATS Creds

Run in the governor-api devcontainer:

```sh
cat /tmp/user.creds
```

Then create and copy into `gov-okta-addon/user.local.creds`

#### Hydra

```sh
GOA_GOVERNOR_CLIENT_SECRET="$(openssl rand -hex 16)"

hydra clients create \
    --endpoint http://hydra:4445/ \
    --audience http://api:3001/ \
    --id gov-okta-addon-governor \
    --secret ${GOA_GOVERNOR_CLIENT_SECRET} \
    --grant-types client_credentials \
    --response-types token,code \
    --token-endpoint-auth-method client_secret_post \
    --scope write,create:governor:users,update:governor:users,read:governor:users,read:governor:groups,read:governor:organizations

# Copy this secret for later
echo $GOA_GOVERNOR_CLIENT_SECRET
```

#### Env

Export the following in the terminal where you will run gov-okta-addon:

```sh
export GOA_NATS_URL="nats://127.0.0.1:4222"
export GOA_OKTA_NOCACHE=true
export GOA_OKTA_URL="https://equinixmetal.oktapreview.com"
export GOA_GOVERNOR_URL="http://127.0.0.1:3001"
export GOA_GOVERNOR_AUDIENCE="http://api:3001/"
export GOA_GOVERNOR_TOKEN_URL="http://127.0.0.1:4444/oauth2/token"
export GOA_GOVERNOR_CLIENT_ID="gov-okta-addon-governor"
export GOA_NATS_CREDS_FILE="${PWD}/user.local.creds"
```

Similarly, ensure you have the following secrets exported:

```sh
# Get from Okta preview app
export GOA_OKTA_TOKEN="REPLACE"
# Secret copied from earlier
export GOA_GOVERNOR_CLIENT_SECRET="REPLACE"
```

#### Troubleshooting

**"error": "Unable to insert or update resource because a resource with that value exists already"**

Run `hydra clients delete gov-okta-addon-governor` in the governor-api devcontainer. Then rerun the steps for hydra.

**"error": "error",**
**"error_description": "The error is unrecognizable"**

Same as above.

### Testing addon serve locally

__WARNING:__ Be careful when running this addon locally - don't point it to the production Okta URL and and
don't run it without `--dry-run` as it could potentially update or wipe out existing groups/users in Okta!

Create a local audit log for testing in the `gov-okta-addon` directory:

```sh
touch audit.log
```

Start the addon (adjust the flags as needed):

```sh
go run . serve --audit-log-path=audit.log --pretty --debug --dry-run
```

### Testing addon sync locally

Run the user sync (adjust the flags as needed):

```sh
go run . sync users --pretty --debug --dry-run
```

You can run the groups and members sync in the same way.
