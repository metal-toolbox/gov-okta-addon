package cmd

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/gosimple/slug"
	okt "github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.equinixmetal.net/gov-okta-addon/internal/governor"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/oauth2/clientcredentials"
)

// syncGroupsCmd syncs okta groups into governor
var syncGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "sync okta groups into governor",
	Long: `Performs a one-way group sync from Okta to Governor.
Groups that exist in Okta but not in Governor, will be created. Groups that exist in Governor but not in Okta, will be deleted.
This command is intended for doing an initial load of groups. It is strongly recommended that you use the dry-run flag first 
to see what groups would be created/deleted in Governor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncGroupsToGovernor(cmd.Context())
	},
}

func init() {
	syncCmd.AddCommand(syncGroupsCmd)

	syncCmd.PersistentFlags().Bool("skip-okta-update", false, "do not make changes to okta groups (ie. setting the governor_id)")
	viperBindFlag("sync.skip-okta-update", syncCmd.PersistentFlags().Lookup("skip-okta-update"))

	syncCmd.PersistentFlags().String("selector-prefix", "", "if set, only group names that start with this string will be processed")
	viperBindFlag("sync.selector-prefix", syncCmd.PersistentFlags().Lookup("selector-prefix"))
}

func syncGroupsToGovernor(ctx context.Context) error {
	dryRun := viper.GetBool("sync.dryrun")
	selectorPrefix := viper.GetString("sync.selector-prefix")

	logger.Info("starting sync to governor", zap.Bool("dry-run", dryRun))

	oc, err := okta.NewClient(
		okta.WithLogger(logger.Desugar()),
		okta.WithURL(viper.GetString("okta.url")),
		okta.WithToken(viper.GetString("okta.token")),
		okta.WithCache((!viper.GetBool("okta.nocache"))),
	)
	if err != nil {
		return err
	}

	gc, err := governor.NewClient(
		governor.WithLogger(logger.Desugar()),
		governor.WithURL(viper.GetString("governor.url")),
		governor.WithClientCredentialConfig(&clientcredentials.Config{
			ClientID:       viper.GetString("governor.client-id"),
			ClientSecret:   viper.GetString("governor.client-secret"),
			TokenURL:       viper.GetString("governor.token-url"),
			EndpointParams: url.Values{"audience": {viper.GetString("governor.audience")}},
			Scopes: []string{
				"write",
				"read:governor:groups",
				"read:governor:organizations",
			},
		}),
	)
	if err != nil {
		return err
	}

	var created, skipped int

	govOrgs, err := govOrgsMap(ctx, gc)
	if err != nil {
		return err
	}

	syncFunc := func(ctx context.Context, g *okt.Group) (*okt.Group, error) {
		l := logger.Desugar().With(zap.String("okta.group.id", g.Id))

		if g.Profile == nil {
			return nil, okta.ErrNilGroupProfile
		}

		groupName := g.Profile.Name
		groupDesc := g.Profile.Description

		l = l.With(zap.String("okta.group.name", groupName))

		if !strings.HasPrefix(strings.ToLower(groupName), strings.ToLower(selectorPrefix)) {
			l.Info("skipping non-selected group")

			skipped++

			return nil, nil
		}

		l.Debug("processing okta group")

		governorID, err := okta.GroupGovernorID(g)
		if err != nil {
			// bail on this group if the error is something other than a not found
			if !errors.Is(err, okta.ErrGroupGovernorIDNotFound) {
				return nil, err
			}
		}

		govGroup, found, err := groupFromGroupID(ctx, gc, governorID, l)
		if err != nil {
			return nil, err
		}

		if govGroup == nil {
			govGroup, err = groupFromGroupSlug(ctx, gc, slug.Make(groupName), l)
			if err != nil {
				return nil, err
			}
		}

		if govGroup == nil {
			l.Info("group not found in governor, creating")

			if !dryRun {
				var err error

				govGroup, err = gc.CreateGroup(ctx, &v1alpha1.GroupReq{
					Name:        groupName,
					Description: groupDesc,
				})
				if err != nil {
					return nil, err
				}

				l = l.With(
					zap.String("governor.group.id", govGroup.ID),
					zap.String("governor.group.slug", govGroup.Slug),
				)

				l.Debug("created governor group from okta sync")
			}

			created++
		}

		// if we found the group by slug or if we created the group, we should update the okta
		// group profile to contain the correct governor id
		if !found {
			grp, err := updateOktaGroupProfile(ctx, oc, g.Id, groupName, groupDesc, govGroup, l)
			if err != nil {
				return nil, err
			}

			g = grp
		}

		apps, err := oc.GroupGithubCloudApplications(ctx, g.Id)
		if err != nil {
			return nil, err
		}

		l.Debug("okta github applications assigned to group", zap.Any("okta.applications", apps))

		if !dryRun {
			govExpectedOrganizations, err := linkGovernorGroupOrganizations(ctx, gc, apps, govGroup, govOrgs, l)
			if err != nil {
				l.Warn("failed to link governor group organizations")
				return nil, err
			}

			l.Debug("pruning orphaned governor group organization assignments")

			if err := pruneOrphanGovernorGroupOrganizations(ctx, gc, govGroup.ID, govExpectedOrganizations, govGroup.Organizations, l); err != nil {
				l.Warn("failed to unlink orphaned governor group organizations")
				return nil, err
			}
		}

		return g, nil
	}

	groups, err := oc.ListGroupsWithModifier(ctx, syncFunc, &query.Params{})
	if err != nil {
		return err
	}

	logger.Desugar().Debug("groups from okta", zap.Any("okta.groups", groups))

	deleted, err := deleteOrphanGovernorGroups(ctx, gc, uniqueGovernorGroupIDs(groups), logger.Desugar())
	if err != nil {
		return err
	}

	logger.Desugar().Info("completed group sync",
		zap.Int("governor.groups.created", created),
		zap.Int("governor.groups.deleted", len(deleted)),
		zap.Int("governor.groups.skipped", skipped),
	)

	return nil
}

// govOrgMaps returns a list of governor org names to
func govOrgsMap(ctx context.Context, gc *governor.Client) (map[string]*v1alpha1.Organization, error) {
	resp := map[string]*v1alpha1.Organization{}

	orgs, err := gc.Organizations(ctx)
	if err != nil {
		return nil, err
	}

	for _, org := range orgs {
		resp[org.Slug] = org
	}

	return resp, nil
}

func linkGovernorGroupOrganizations(
	ctx context.Context,
	gc *governor.Client,
	oktaApps map[string]string,
	govGroup *v1alpha1.Group,
	govOrgs map[string]*v1alpha1.Organization,
	l *zap.Logger) ([]string, error) {
	govExpectedOrganizations := []string{}

	// loop over the okta github applications to get the organization
	// and ensure the governor group is linked with appropriate governor orgs
	for orgName := range oktaApps {
		// ensure governor manages the org, otherwise skip over it
		org, ok := govOrgs[orgName]
		if !ok {
			l.Warn("assigned application org doesn't exist as a governor organization",
				zap.String("okta.application.org", orgName),
			)

			continue
		}

		l := l.With(
			zap.String("governor.org.id", org.ID),
			zap.String("governor.org.name", org.Name),
		)

		govExpectedOrganizations = append(govExpectedOrganizations, org.ID)

		l.Debug("found governor organization associated with the okta application",
			zap.String("governor.org.id", org.ID),
			zap.String("governor.org.name", org.Name),
		)

		if !contains(govGroup.Organizations, org.ID) {
			l.Info("linking governor group to organization",
				zap.String("governor.org.id", org.ID),
				zap.String("governor.org.name", org.Name),
			)

			if err := gc.AddGroupToOrganization(ctx, govGroup.ID, org.ID); err != nil {
				l.Warn("failed to add governor group to organization",
					zap.String("governor.org.id", org.ID),
					zap.String("governor.org.name", org.Name),
				)

				continue
			}
		}
	}

	return govExpectedOrganizations, nil
}

func pruneOrphanGovernorGroupOrganizations(ctx context.Context, gc *governor.Client, groupID string, expected, actual []string, l *zap.Logger) error {
	// remove any organization links that are unexpected
	for _, org := range actual {
		if !contains(expected, org) {
			l.Info("unexpected governor organization link, removing from governor group",
				zap.String("governor.org.id", org),
			)

			if err := gc.RemoveGroupFromOrganization(ctx, groupID, org); err != nil {
				l.Warn("failed to remove governor group to organization",
					zap.String("governor.org.id", org),
				)

				continue
			}
		}
	}

	return nil
}

// groupFromGroupID gets a governor group from a group id (presumably from an okta group profile).  If the groupID is
// empty or the group is not found, return a nil group.  Otherwise, return the group we got back from governor.
func groupFromGroupID(ctx context.Context, gc *governor.Client, groupID string, l *zap.Logger) (*v1alpha1.Group, bool, error) {
	if groupID == "" {
		return nil, false, nil
	}

	l.Debug("getting group from governor")

	// if we have a governor ID from okta, try to get the group in governor
	govGroup, err := gc.Group(ctx, groupID)
	if err != nil {
		// bail on governor errors other than group not found
		if !errors.Is(err, governor.ErrGroupNotFound) {
			return nil, false, err
		}

		l.Warn("governor id found on okta group, but group not found in governor",
			zap.String("governor.id", groupID),
		)

		return nil, false, nil
	}

	l = l.With(
		zap.String("governor.group.id", govGroup.ID),
		zap.String("governor.group.slug", govGroup.Slug),
	)

	l.Debug("group id exists in governor")

	return govGroup, true, nil
}

// groupFromGroupSlug gets a governor group from a group slug (presumably generated from an okta group name).  If the
// group slug is empty or the group is not found, return a nil group.  Otherwise, return the group we got back from governor.
func groupFromGroupSlug(ctx context.Context, gc *governor.Client, slug string, l *zap.Logger) (*v1alpha1.Group, error) {
	if slug == "" {
		return nil, nil
	}

	l.Debug("getting group from governor")

	// if we have a governor slug, try to get the group in governor
	govGroup, err := gc.Group(ctx, slug)
	if err != nil {
		// bail on governor errors other than group not found
		if !errors.Is(err, governor.ErrGroupNotFound) {
			return nil, err
		}

		l.Warn("group slug not found in governor", zap.String("governor.id", slug))

		return nil, nil
	}

	l = l.With(
		zap.String("governor.group.id", govGroup.ID),
		zap.String("governor.group.slug", govGroup.Slug),
	)

	l.Debug("group slug exists in governor")

	return govGroup, nil
}

func deleteOrphanGovernorGroups(ctx context.Context, gc *governor.Client, gIDs map[string]struct{}, l *zap.Logger) ([]string, error) {
	dryRun := viper.GetBool("sync.dryrun")
	selectorPrefix := viper.GetString("sync.selector-prefix")

	groups, err := gc.Groups(ctx)
	if err != nil {
		return nil, err
	}

	deleted := []string{}

	for _, group := range groups {
		if _, ok := gIDs[group.ID]; !ok {
			if !strings.HasPrefix(strings.ToLower(group.Slug), strings.ToLower(selectorPrefix)) {
				l.Debug("skipping delete of non-selected group",
					zap.String("governor.group.id", group.ID),
					zap.String("governor.group.name", group.Name),
					zap.String("governor.group.slug", group.Slug),
				)

				continue
			}

			l.Info("deleting orphaned group from governor",
				zap.String("governor.group.id", group.ID),
				zap.String("governor.group.name", group.Name),
				zap.String("governor.group.slug", group.Slug),
			)

			if !dryRun {
				if err := gc.DeleteGroup(ctx, group.ID); err != nil {
					l.Warn("failed to delete orphaned governor group",
						zap.String("governor.group.id", group.ID),
						zap.String("governor.group.name", group.Name),
						zap.String("governor.group.slug", group.Slug),
						zap.Error(err),
					)

					continue
				}
			}

			deleted = append(deleted, group.Slug)
		}
	}

	return deleted, nil
}

// uniqueGovernorGroupIDs returns a map of unique governor ids from a slice of okta groups
func uniqueGovernorGroupIDs(groups []*okt.Group) map[string]struct{} {
	l := logger.Desugar()

	resp := map[string]struct{}{}

	for _, g := range groups {
		govID, err := okta.GroupGovernorID(g)
		if err != nil {
			l.Warn("unable to get governor id from okta group", zap.Error(err))
			continue
		}

		l.Debug("found governor group id", zap.String("governor.group.id", govID))

		resp[govID] = struct{}{}
	}

	return resp
}

// updateOktaGroupProfile returns a fake updated profile if we're skipping okta updates
// or updates the profile in okta and returns the real thing
func updateOktaGroupProfile(
	ctx context.Context,
	oc *okta.Client,
	gID, groupName, groupDesc string,
	govGroup *v1alpha1.Group,
	l *zap.Logger) (*okt.Group, error) {
	skipOkta := viper.GetBool("sync.skip-okta-update")
	dryRun := viper.GetBool("sync.dryrun")

	if skipOkta || dryRun {
		l.Info("skipping okta update of governor id")

		grp := &okt.Group{
			Id: gID,
			Profile: &okt.GroupProfile{
				Name:            groupName,
				Description:     groupDesc,
				GroupProfileMap: okt.GroupProfileMap{},
			},
		}

		if dryRun {
			grp.Profile.GroupProfileMap[okta.GroupProfileGovernorIDKey] = "FAKE"
			return grp, nil
		}

		grp.Profile.GroupProfileMap[okta.GroupProfileGovernorIDKey] = govGroup.ID

		return grp, nil
	}

	l.Info("writing governor id on okta group profile")

	group, err := oc.UpdateGroup(
		ctx,
		gID,
		groupName,
		groupDesc,
		map[string]interface{}{okta.GroupProfileGovernorIDKey: govGroup.ID},
	)
	if err != nil {
		return nil, err
	}

	return group, err
}
