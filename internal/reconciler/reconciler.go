package reconciler

import (
	"context"
	"errors"
	"time"

	"go.equinixmetal.net/gov-okta-addon/internal/governor"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha"
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

// Run starts the reconciler
func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("starting reconciler loop", zap.Duration("interval", r.interval))

	for {
		select {
		case <-ticker.C:
			r.logger.Debug("executing reconciler loop",
				zap.String("time", time.Now().UTC().Format(time.RFC3339)),
			)

			groups, err := r.governorClient.Groups(ctx)
			if err != nil {
				r.logger.Error("error listing group", zap.Error(err))
				continue
			}

			r.logger.Debug("got groups response", zap.Any("groups list", groups))

			oktaAppOrgs, err := r.oktaClient.GithubCloudApplications(ctx)
			if err != nil {
				r.logger.Error("error listing okta githubcloud applications", zap.Error(err))
				continue
			}

			r.logger.Debug("got okta github cloud orgs", zap.Any("github.orgs", oktaAppOrgs))

			for _, g := range groups {
				if err := r.reconcileGroup(ctx, g, oktaAppOrgs); err != nil {
					r.logger.Error("error reconciling governor group", zap.String("group.id", g.ID), zap.String("group.slug", g.Slug))
					continue
				}
			}
		case <-ctx.Done():
			r.logger.Info("shutting down reconciler",
				zap.String("time", time.Now().UTC().Format(time.RFC3339)),
			)

			return
		}
	}
}

// reconcileGroup reconciles a governor group with an okta group
func (r *Reconciler) reconcileGroup(ctx context.Context, g *v1alpha.Group, oktaAppOrgs map[string]string) error {
	logger := r.logger.With(zap.String("group.id", g.ID), zap.String("group.slug", g.Slug))

	group, err := r.governorClient.Group(ctx, g.ID)
	if err != nil {
		logger.Error("error getting governor group", zap.Error(err))
		return err
	}

	// TODO remove this once we're more comfortable with governor managing okta groups
	// use assigned organizations in governor to trigger creating groups in Okta
	if len(group.Organizations) == 0 {
		logger.Info("skipping group without any organizations assigned")
		return nil
	}

	logger.Debug("got group response", zap.Any("group details", group))

	oktaGroup, err := r.oktaClient.GetGroupByGovernorID(ctx, g.ID)
	if err != nil {
		if !errors.Is(err, okta.ErrGroupsNotFound) {
			logger.Error("error getting okta group by governor id", zap.Error(err))
			return err
		}

		ogID, err := r.oktaClient.CreateGroup(ctx, g.Name, g.Description, map[string]interface{}{"governor_id": g.ID})
		if err != nil {
			logger.Error("error creating okta group", zap.Error(err))
			return err
		}

		logger.Info("created okta group", zap.String("okta.group.id", ogID))
	}

	logger.Debug("got okta group", zap.Any("okta.group", oktaGroup))

	for _, o := range group.Organizations {
		org, err := r.governorClient.Organization(ctx, o)
		if err != nil {
			logger.Error("error getting organization from governor", zap.Error(err))
			return err
		}

		logger.Debug("got governor organization", zap.Any("governor.org", org))

		oktaAppID, ok := oktaAppOrgs[org.Name]
		if !ok {
			logger.Warn("assigned organization not found in okta github applications", zap.String("governor.org", org.Name))
			continue
		}

		assignments, err := r.oktaClient.ListGroupApplicationAssignment(ctx, oktaAppID)
		if err != nil {
			logger.Error("error listing okta group assigned to okta application", zap.String("okta.app", oktaAppID))
			return err
		}

		logger.Debug("list of groups for application", zap.Any("groups", assignments))

		if contains(assignments, oktaGroup) {
			continue
		}

		// if err := r.oktaClient.GetGroupApplicationAssignment(ctx, oktaAppID, oktaGroup); err != nil {
		// 	logger.Error("error getting okta group to okta application assignment", zap.String("okta.app", oktaAppID), zap.String("okta.group.id", oktaGroup))
		// 	return err
		// }

		if err := r.oktaClient.AssignGroupToApplication(ctx, oktaAppID, oktaGroup); err != nil {
			logger.Error("error assigning okta group to okta application", zap.String("okta.app", oktaAppID), zap.String("okta.group.id", oktaGroup))
			return err
		}
	}

	return nil
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}

	return false
}
