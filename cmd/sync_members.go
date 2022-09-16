package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.equinixmetal.net/gov-okta-addon/internal/governor"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	userCache = make(map[string]*v1alpha1.User)
)

// syncGroupsCmd syncs okta groups into governor
var syncMembershipCmd = &cobra.Command{
	Use:   "membership",
	Short: "sync okta group membership into governor",
	Long: `Performs a one-way group membership sync from Okta to Governor.
Group members that exist in Okta, but not in Governor, will be added to the Governor group.  Governor
group members that do not exist in the Okta group will be removed from the group.  Groups and Users
must exist in Governor. It is strongly recommended that you use the dry-run flag first to see what
group memberships would be created/deleted in Governor.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncGroupMembershipToGovernor(cmd.Context())
	},
}

func init() {
	syncCmd.AddCommand(syncMembershipCmd)
}

func syncGroupMembershipToGovernor(ctx context.Context) error {
	logger := logger.Desugar()

	dryRun := viper.GetBool("sync.dryrun")

	logger.Info("starting sync to governor", zap.Bool("dry-run", dryRun))

	oc, err := okta.NewClient(
		okta.WithLogger(logger),
		okta.WithURL(viper.GetString("okta.url")),
		okta.WithToken(viper.GetString("okta.token")),
		okta.WithCache((!viper.GetBool("okta.nocache"))),
	)
	if err != nil {
		return err
	}

	gc, err := governor.NewClient(
		governor.WithLogger(logger),
		governor.WithURL(viper.GetString("governor.url")),
		governor.WithClientCredentialConfig(&clientcredentials.Config{
			ClientID:       viper.GetString("governor.client-id"),
			ClientSecret:   viper.GetString("governor.client-secret"),
			TokenURL:       viper.GetString("governor.token-url"),
			EndpointParams: url.Values{"audience": {viper.GetString("governor.audience")}},
			Scopes: []string{
				"write",
				"read:governor:groups",
			},
		}),
	)
	if err != nil {
		return err
	}

	govGroups, err := gc.Groups(ctx)
	if err != nil {
		return err
	}

	logger.Debug("processing list of governor groups", zap.Int("governor.groups.count", len(govGroups)))

	var updatedGroups, skippedGroups, skippedUsers int

	for _, g := range govGroups {
		l := logger.With(
			zap.String("governor.group.id", g.ID),
			zap.String("governor.group.slug", g.Slug),
		)

		// get the details of the governor group (including the membership)
		govGroup, err := gc.Group(ctx, g.ID)
		if err != nil {
			return err
		}

		l.Debug("got governor group details", zap.Any("governor.group", govGroup))

		// get the okta group from the governor id
		oktaGroupID, err := oc.GetGroupByGovernorID(ctx, govGroup.ID)
		if err != nil {
			if errors.Is(err, okta.ErrGroupsNotFound) {
				l.Info("governor group not found in okta, skipping")

				skippedGroups++

				continue
			}

			l.Error("failed to get okta group by governor id", zap.Error(err))

			return err
		}

		l = l.With(zap.String("okta.group.id", oktaGroupID))

		// get the list of users on the okta group
		oktaGroupMembership, err := oc.ListGroupMembership(ctx, oktaGroupID)
		if err != nil {
			l.Error("failed to list okta group membership", zap.Error(err))
			return err
		}

		l.Debug("got okta group membership", zap.Strings("okta.group.members", oktaGroupMembership))

		expectedMembers := []string{}

		for _, oktaMemberUserID := range oktaGroupMembership {
			user, err := governorUserFromOktaID(ctx, gc, oktaMemberUserID, l)
			if err != nil {
				l.Error("failed to query governor for user, skipping",
					zap.String("okta.user.id", oktaMemberUserID),
					zap.Error(err),
				)

				skippedUsers++

				continue
			}

			expectedMembers = append(expectedMembers, user.ID)

			if !contains(govGroup.Members, user.ID) {
				// TODO add user to group in governor
				l.Info("adding user to governor group",
					zap.String("goveror.user.id", user.ID),
					zap.String("goveror.user.email", user.Email),
					zap.String("goveror.user.external_id", user.ExternalID),
					zap.String("okta.user.id", oktaMemberUserID),
				)
			}
		}

		for _, m := range govGroup.Members {
			if !contains(expectedMembers, m) {
				// TODO remove user from group in governor
				l.Info("pruning user from governor group",
					zap.String("goveror.user.id", m),
				)
			}
		}
	}

	logger.Info("completed group sync",
		zap.Int("governor.groups.updated", updatedGroups),
		zap.Int("governor.groups.skipped", skippedGroups),
	)

	return nil
}

func governorUserFromOktaID(ctx context.Context, gc *governor.Client, oktaID string, l *zap.Logger) (*v1alpha1.User, error) {
	// get the governor user
	user, ok := userCache[oktaID]
	if !ok {
		u, err := gc.UsersQuery(ctx, map[string][]string{"external_id": {oktaID}})
		if err != nil {
			return nil, err
		}

		if count := len(u); count != 0 {
			return nil, fmt.Errorf("unexpected user count from query: %d expected 1", count)
		}

		userCache[oktaID] = u[0]
		user = u[0]
	}

	return user, nil
}
