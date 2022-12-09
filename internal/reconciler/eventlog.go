package reconciler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	okt "go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

var (
	// defaultPollerInterval is the default for how often to poll for new events
	defaultPollerInterval = 30 * time.Second
	// defaultColdStartLookback is the default for how far back to go for events on a cold start
	defaultColdStartLookback = 6 * time.Hour
)

func (r *Reconciler) startEventLogPollerSubscriptions(ctx context.Context) {
	r.logger.Debug("starting event log polling")
	r.oktaClient.PollLogs(
		ctx,
		defaultPollerInterval,
		time.Now().UTC().Add(-defaultColdStartLookback),
		&query.Params{
			Filter: "eventType eq \"user.lifecycle.create\"",
		},
		r.oktaLogEventHandler)
}

func (r *Reconciler) oktaLogEventHandler(ctx context.Context, evt *okta.LogEvent) {
	r.logger.Debug("handling event from okta log", zap.String("okta.event.type", evt.EventType), zap.Any("okta.event", evt))

	switch evt.EventType {
	case "user.lifecycle.create":
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

			logger := r.logger.With(zap.String("okta.user.id", oktUser.Id), zap.String("okta.user.email", email))

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

			logger.Debug("got users from governor by email", zap.Any("governor.users", govUsers))

			switch len(govUsers) {
			case 0:
				logger.Debug("okta user does not exist in governor, creating")

				if !r.dryrun {
					govUser, err := r.governorClient.CreateUser(ctx, &v1alpha1.UserReq{
						Email:      email,
						ExternalID: oktUser.Id,
						Name:       fmt.Sprintf("%s %s", first, last),
						Status:     "active",
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

				if govUser.Status.String != "pending" && govUser.ExternalID.String != "" {
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
						Status:     "active",
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
	default:
		r.logger.Warn("unhandled okta event type", zap.String("okta.event.type", evt.EventType))
	}
}
