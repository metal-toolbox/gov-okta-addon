package reconciler

import (
	"context"
	"errors"
	"time"

	"go.equinixmetal.net/gov-okta-addon/internal/governor"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"

	"go.uber.org/zap"
)

const (
	defaultReconcileInterval = 1 * time.Hour
)

// Reconciler reconciles Governor groups/users with Okta
type Reconciler struct {
	interval       time.Duration
	governorClient *governor.Client
	logger         *zap.Logger
	oktaClient     *okta.Client
	dryrun         bool
	skipDelete     bool
}

// Option is a functional configuration option
type Option func(r *Reconciler)

// WithInterval sets the reconciler interval
func WithInterval(i time.Duration) Option {
	return func(r *Reconciler) {
		r.interval = i
	}
}

// WithLogger sets logger
func WithLogger(l *zap.Logger) Option {
	return func(r *Reconciler) {
		r.logger = l
	}
}

// WithDryRun sets dryrun
func WithDryRun(d bool) Option {
	return func(r *Reconciler) {
		r.dryrun = d
	}
}

// WithSkipDelete sets skipDelete
func WithSkipDelete(s bool) Option {
	return func(r *Reconciler) {
		r.skipDelete = s
	}
}

// WithOktaClient sets okta client
func WithOktaClient(o *okta.Client) Option {
	return func(r *Reconciler) {
		r.oktaClient = o
	}
}

// WithGovernorClient sets governor api client
func WithGovernorClient(c *governor.Client) Option {
	return func(r *Reconciler) {
		r.governorClient = c
	}
}

// New returns a new reconciler
func New(opts ...Option) *Reconciler {
	rec := Reconciler{
		logger:   zap.NewNop(),
		interval: defaultReconcileInterval,
	}

	for _, opt := range opts {
		opt(&rec)
	}

	rec.logger.Debug("creating new reconciler")

	return &rec
}

// Run starts the reconciler.  The reconciler loop:
//   - gets the full list of groups from governor
//   - ensures each of those groups exist in okta
//   - assigns github applications to those groups in okta for
//     each organization associated with the group
func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("starting reconciler loop", zap.Duration("interval", r.interval), zap.Bool("dryrun", r.dryrun), zap.Bool("skip-delete", r.skipDelete))

	for {
		select {
		case <-ticker.C:
			r.logger.Info("executing reconciler loop",
				zap.String("time", time.Now().UTC().Format(time.RFC3339)),
			)

			groups, err := r.governorClient.Groups(ctx)
			if err != nil {
				r.logger.Error("error listing group", zap.Error(err))
				continue
			}

			r.logger.Debug("got groups response", zap.Any("groups list", groups))

			// collect a map of okta group ids to governor groups so we don't have to
			// go back to the okta API for this data and risk getting throttled
			groupMap := map[string]*v1alpha1.Group{}

			for _, g := range groups {
				logger := r.logger.With(zap.String("governor.group.id", g.ID), zap.String("governor.group.slug", g.Slug))

				groupDetails, err := r.governorClient.Group(ctx, g.ID)
				if err != nil {
					logger.Error("error getting governor group details", zap.Error(err))
					continue
				}

				logger.Debug("got governor group response", zap.Any("group details", groupDetails))

				oktaGroupID, err := r.groupExists(ctx, g.ID)
				if err != nil {
					logger.Error("error reconciling governor group exists")
					continue
				}

				groupMap[oktaGroupID] = groupDetails

				if err := r.GroupMembership(ctx, g.ID, oktaGroupID); err != nil {
					logger.Error("error reconciling governor group membership")
					continue
				}
			}

			if err := r.reconcileGroupApplicationAssignments(ctx, groupMap); err != nil {
				r.logger.Error("error reconciling group application links", zap.Error(err))
			}

			// reconcile users

			govUsers, err := r.governorClient.Users(ctx, true)
			if err != nil {
				r.logger.Error("error listing governor users", zap.Error(err))
			}

			r.logger.Debug("got governor users (including deleted)", zap.Any("num.governor.users", len(govUsers)))

			oktaUsers, err := r.oktaClient.ListUsers(ctx)
			if err != nil {
				r.logger.Error("error listing okta users", zap.Error(err))
			}

			oktaUserIDs := map[string]struct{}{}

			for _, oktaUser := range oktaUsers {
				oktaUserIDs[oktaUser.Id] = struct{}{}
			}

			r.logger.Debug("got okta users", zap.Any("okta.user.ids", oktaUserIDs))

			if err := r.reconcileUsers(ctx, govUsers, oktaUserIDs); err != nil {
				r.logger.Error("error reconciling users", zap.Error(err))
			}

		case <-ctx.Done():
			r.logger.Info("shutting down reconciler",
				zap.String("time", time.Now().UTC().Format(time.RFC3339)),
			)

			return
		}
	}
}

// reconcileGroupApplicationAssignments reconciles the application assignments for all groups.  It takes a map
// of okta group ids to governor groups and does it's best to make as few calls to okta as possible to prevent
// throttling.  A call to this function without any changes will result in n+1 calls to the Okta API where
// n is the number of Okta github cloud applications.
func (r *Reconciler) reconcileGroupApplicationAssignments(ctx context.Context, groupMap map[string]*v1alpha1.Group) error {
	// get the github cloud apps first from okta
	oktaAppOrgs, err := r.oktaClient.GithubCloudApplications(ctx)
	if err != nil {
		r.logger.Error("error listing okta github cloud applications", zap.Error(err))
		return err
	}

	r.logger.Debug("got okta github cloud orgs", zap.Any("github.orgs", oktaAppOrgs))

	govOrgs, err := r.governorClient.Organizations(ctx)
	if err != nil {
		r.logger.Error("error listing governor organizations", zap.Error(err))
		return err
	}

	r.logger.Debug("got governor organizations", zap.Any("governor.orgs", govOrgs))

	// for each of the okta github cloud applications, get the groups assigned to the application
	for org, appID := range oktaAppOrgs {
		logger := r.logger.With(zap.String("okta.app.org", org), zap.String("okta.app.id", appID))

		assignments, err := r.oktaClient.ListGroupApplicationAssignment(ctx, appID)
		if err != nil {
			logger.Error("error listing okta group assigned to okta application")
			return err
		}

		logger.Debug("list of groups for application", zap.Any("groups", assignments))

		// foreach governor/okta group, check if should be assigned to the app and reconcile
		for oktaGID, groupDetails := range groupMap {
			logger := logger.With(
				zap.String("governor.group.id", groupDetails.ID),
				zap.String("governor.group.slug", groupDetails.Slug),
				zap.String("okta.group.id", oktaGID),
			)

			slugs := getGroupOrgSlugs(groupDetails, govOrgs)

			logger.Debug("got governor group org slugs", zap.Strings("slugs", slugs))

			// if the group organizations contains the github organization for the okta application
			if contains(slugs, org) {
				logger.Debug("group org list contains app org slug, ensuring group is assigned to okta app")

				// ensure it exists in the app in okta
				if contains(assignments, oktaGID) {
					continue
				}

				// assign group to the application
				if r.dryrun {
					logger.Info("SKIP assigning okta group to okta application", zap.String("okta.app.id", appID))
					continue
				}

				if err := r.oktaClient.AssignGroupToApplication(ctx, appID, oktaGID); err != nil {
					logger.Error("error assigning okta group to okta application", zap.String("okta.app.id", appID))
					return err
				}

				groupsApplicationAssignedCounter.Inc()

				continue
			}

			logger.Debug("group org list does not contain app org slug, ensuring group is not assigned to okta app")

			// ensure it doesn't exist in the okta app
			if !contains(assignments, oktaGID) {
				continue
			}

			// remove group from the application
			if r.dryrun || r.skipDelete {
				logger.Info("SKIP removing assignment of okta group from okta application", zap.String("okta.app.id", appID))
			} else {
				if err := r.oktaClient.RemoveApplicationGroupAssignment(ctx, appID, oktaGID); err != nil {
					logger.Error("error removing okta group from okta application", zap.String("okta.app.id", appID))
					return err
				}

				groupsApplicationUnassignedCounter.Inc()
			}
		}
	}

	return nil
}

// reconcileUsers gets a list of governor users and a map of all user ids from okta, and
// deletes any okta users that have been deleted in governor. Note that we are specifically
// targeting users who have existed in governor and have been deleted, and not just users who
// do not exist in governor
func (r *Reconciler) reconcileUsers(ctx context.Context, govUsers []*v1alpha1.User, oktaUserIDs map[string]struct{}) error {
	if govUsers == nil || oktaUserIDs == nil {
		return ErrUserListEmpty
	}

	r.logger.Debug("reconciling users")

	for _, u := range govUsers {
		if !userDeleted(u) {
			// active user in governor, skip
			continue
		}

		logger := r.logger.With(
			zap.String("governor.user.id", u.ID),
			zap.String("okta.user.id", u.ExternalID),
			zap.String("governor.user.email", u.Email),
		)

		logger.Debug("got deleted governor user")

		// user has been deleted in governor, so delete it in okta if still there
		if _, found := oktaUserIDs[u.ExternalID]; found {
			if r.dryrun || r.skipDelete {
				logger.Info("SKIP deleting okta user")
				continue
			}
			// TODO: re-enable when we feel confident, or when we dry-run
			// if err := r.oktaClient.DeleteUser(ctx, u.ExternalID); err != nil {
			// 	logger.Error("error deleting user", zap.Error(err))
			// 	continue
			// }
			//
			// logger.Info("successfully deleted okta user")
			logger.Debug("skipping user deletion in okta")
		} else {
			logger.Debug("user not found in okta")
		}
	}

	return nil
}

// groupExists ensures the governor group exists in okta
func (r *Reconciler) groupExists(ctx context.Context, id string) (string, error) {
	logger := r.logger.With(zap.String("governor.group.id", id))

	oktaGroup, err := r.oktaClient.GetGroupByGovernorID(ctx, id)
	if err != nil {
		if !errors.Is(err, okta.ErrGroupsNotFound) {
			logger.Error("error getting okta group by governor id", zap.Error(err))
			return "", err
		}

		oktaGID, err := r.GroupCreate(ctx, id)
		if err != nil {
			return "", err
		}

		return oktaGID, nil
	}

	logger.Debug("got okta group", zap.Any("okta.group", oktaGroup))

	return oktaGroup, nil
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}

	return false
}
