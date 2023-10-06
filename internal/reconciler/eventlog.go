package reconciler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	okt "github.com/metal-toolbox/gov-okta-addon/internal/okta"
	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

var (
	// DefaultEventlogPollerInterval is the default for how often to poll for new events
	DefaultEventlogPollerInterval = 30 * time.Second
	// DefaultEventlogColdStartLookback is the default for how far back to go for events on a cold start
	DefaultEventlogColdStartLookback = 8 * time.Hour
)

func (r *Reconciler) startEventLogPollerSubscriptions(ctx context.Context) {
	r.logger.Debug("starting okta event log polling")
	r.oktaClient.PollLogs(
		ctx,
		r.eventlogInterval,
		time.Now().UTC().Add(-r.eventlogLookback),
		&query.Params{
			// https://developer.okta.com/docs/reference/core-okta-api/#filter
			Filter: `(eventType eq "user.lifecycle.create" or eventType eq "user.lifecycle.suspend" or eventType eq "user.lifecycle.unsuspend")`,
		},
		r.oktaLogEventHandler)
}

func (r *Reconciler) oktaLogEventHandler(ctx context.Context, evt *okta.LogEvent) {
	r.logger.Debug("handling event from okta log", zap.String("okta.event.type", evt.EventType), zap.Any("okta.event", evt))

	switch evt.EventType {
	case "user.lifecycle.create":
		r.userLifecycleCreateHandler(ctx, evt)

	case "user.lifecycle.suspend", "user.lifecycle.unsuspend":
		r.userLifecycleSuspendHandler(ctx, evt)

	default:
		r.logger.Warn("unhandled okta event type", zap.String("okta.event.type", evt.EventType))
	}
}

// userLifecycleCreateHandler will create a new user in governor if the user does not exist
func (r *Reconciler) userLifecycleCreateHandler(ctx context.Context, evt *okta.LogEvent) {
	for _, target := range evt.Target {
		if target.Type != "User" {
			r.logger.Warn("unexpected target type for user.lifecycle.create", zap.String("okta.event.target.type", target.Type))
			continue
		}

		oktUser, err := r.oktaClient.GetUser(ctx, target.Id)
		if err != nil {
			r.logger.Warn("error getting user from okta", zap.String("okta.user.id", target.Id), zap.Error(err))
			continue
		}

		email, err := okt.EmailFromUserProfile(oktUser)
		if err != nil {
			r.logger.Warn("error getting user email from okta profile", zap.String("okta.user.id", target.Id), zap.Error(err))
			continue
		}

		logger := r.logger.With(
			zap.String("okta.event.type", evt.EventType),
			zap.String("okta.user.id", oktUser.Id),
			zap.String("okta.user.email", email),
		)

		first, err := okt.FirstNameFromUserProfile(oktUser)
		if err != nil {
			logger.Warn("error getting users first name from okta profile")
			continue
		}

		last, err := okt.LastNameFromUserProfile(oktUser)
		if err != nil {
			logger.Warn("error getting users last name from okta profile")
			continue
		}

		govUsers, err := r.governorClient.UsersQuery(ctx, map[string][]string{"email": {email}})
		if err != nil {
			logger.Warn("error getting user by email from governor")
			continue
		}

		logger.Debug("got user(s) from governor by email", zap.Any("governor.users", govUsers))

		switch len(govUsers) {
		case 0:
			logger.Debug("okta user does not exist in governor, creating")

			if !r.dryrun {
				govUser, err := r.governorClient.CreateUser(ctx, &v1alpha1.UserReq{
					Email:      email,
					ExternalID: oktUser.Id,
					Name:       fmt.Sprintf("%s %s", first, last),
					Status:     v1alpha1.UserStatusActive,
				})
				if err != nil {
					logger.Warn("error creating governor user", zap.Error(err))
					continue
				}

				logger.Info("created governor user", zap.String("governor.user.id", govUser.ID))

				continue
			}

			logger.Info("SKIP created governor user")
		case 1:
			govUser := govUsers[0]

			logger.Debug("okta user exists in governor, updating profile", zap.String("governor.user.id", govUser.ID))

			if govUser.Status.String != v1alpha1.UserStatusPending && govUser.ExternalID.String != "" {
				logger.Info("skipping update for user with non-pending status and non-empty external id",
					zap.String("governor.user.status", govUser.Status.String),
					zap.String("governor.user.external_id", govUser.ExternalID.String),
				)

				continue
			}

			if !r.dryrun {
				payload := &v1alpha1.UserReq{
					Email:      email,
					ExternalID: oktUser.Id,
					Name:       fmt.Sprintf("%s %s", first, last),
					Status:     v1alpha1.UserStatusActive,
				}

				logger.Debug("updating governor user with payload", zap.Any("payload", payload))

				govUser, err := r.governorClient.UpdateUser(ctx, govUser.ID, payload)
				if err != nil {
					logger.Warn("error updating governor user", zap.Error(err))
					continue
				}

				logger.Info("updated governor user", zap.String("governor.user.id", govUser.ID))

				continue
			}

			logger.Info("SKIP updated governor user", zap.String("governor.user.id", govUser.ID))

		default:
			logger.Warn("unexpected number of governor users with email, skipping")
			continue
		}
	}
}

// userLifecycleSuspendHandler will suspend or un-suspend a governor user. It does not rely on the lifecycle
// event name but will look up the current user status in okta and update the governor user accordingly.
func (r *Reconciler) userLifecycleSuspendHandler(ctx context.Context, evt *okta.LogEvent) {
	for _, target := range evt.Target {
		if target.Type != "User" {
			r.logger.Warn("unexpected target type for user.lifecycle.create", zap.String("okta.event.target.type", target.Type))
			continue
		}

		oktUser, err := r.oktaClient.GetUser(ctx, target.Id)
		if err != nil {
			r.logger.Warn("error getting user from okta", zap.String("okta.user.id", target.Id), zap.Error(err))
			continue
		}

		details, err := okt.UserDetailsFromOktaUser(oktUser)
		if err != nil {
			r.logger.Warn("error getting user details from okta profile", zap.String("okta.user.id", target.Id), zap.Error(err))
			continue
		}

		logger := r.logger.With(
			zap.String("okta.event.type", evt.EventType),
			zap.String("okta.user.id", oktUser.Id),
			zap.String("okta.user.email", details.Email),
			zap.String("okta.user.status", details.Status),
		)

		govUsers, err := r.governorClient.UsersQuery(ctx, map[string][]string{"email": {details.Email}})
		if err != nil {
			logger.Warn("error getting user by email from governor")
			continue
		}

		logger.Debug("got user(s) from governor by email", zap.Any("governor.users", govUsers))

		switch len(govUsers) {
		case 0:
			logger.Info("okta user not found in governor, skipping")
			continue
		case 1:
			govUser := govUsers[0]

			if govUser.Status.String == v1alpha1.UserStatusPending {
				logger.Info("skipping pending governor user")
				continue
			}

			if details.Status != "SUSPENDED" && details.Status != "ACTIVE" {
				logger.Info("skipping suspend/unsuspend for okta user with unexpected status", zap.String("okta.user.status", details.Status))
				continue
			}

			if govUser.Status.String == v1alpha1.UserStatusActive && details.Status == "SUSPENDED" {
				if !r.dryrun {
					payload := &v1alpha1.UserReq{
						Status: v1alpha1.UserStatusSuspended,
					}

					govUser, err := r.governorClient.UpdateUser(ctx, govUser.ID, payload)
					if err != nil {
						logger.Warn("error suspending governor user", zap.Error(err))
						continue
					}

					logger.Info("suspended governor user", zap.String("governor.user.id", govUser.ID))

					continue
				}

				logger.Info("SKIP suspending governor user", zap.String("governor.user.id", govUser.ID))

				continue
			}

			if govUser.Status.String == v1alpha1.UserStatusSuspended && details.Status == "ACTIVE" {
				if !r.dryrun {
					payload := &v1alpha1.UserReq{
						Status: v1alpha1.UserStatusActive,
					}

					govUser, err := r.governorClient.UpdateUser(ctx, govUser.ID, payload)
					if err != nil {
						logger.Warn("error un-suspending governor user", zap.Error(err))
						continue
					}

					logger.Info("un-suspended governor user", zap.String("governor.user.id", govUser.ID))

					continue
				}

				logger.Info("SKIP un-suspending governor user", zap.String("governor.user.id", govUser.ID))

				continue
			}

			logger.Info("no action needed for user", zap.String("governor.user.status", govUser.Status.String))

		default:
			logger.Warn("unexpected number of governor users with email, skipping")
			continue
		}
	}
}
