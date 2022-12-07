package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	okt "github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	governor "go.equinixmetal.net/governor/pkg/client"

	"go.uber.org/zap"
	"golang.org/x/oauth2/clientcredentials"
)

type memberSummary struct {
	skipped []string
	added   []string
	removed []string
}

var userCache = make(map[string]*v1alpha1.User)

// syncMembersCmd syncs okta groups members into governor
var syncMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "sync okta group membership into governor",
	Long: `Performs a one-way group membership sync from Okta to Governor.
Group members that exist in Okta, but not in Governor, will be added to the Governor group.  Governor
group members that do not exist in the Okta group will be removed from the group.  Groups and Users
must exist in Governor. It is strongly recommended that you use the dry-run flag first to see what
group memberships would be created/deleted in Governor.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncGroupMembersToGovernor(cmd.Context())
	},
}

func init() {
	syncCmd.AddCommand(syncMembersCmd)
}

func syncGroupMembersToGovernor(ctx context.Context) error {
	logger := logger.Desugar()
	dryRun := viper.GetBool("sync.dryrun")

	logger.Info("starting sync to governor group members", zap.Bool("dry-run", dryRun))

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
				"read:governor:users",
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

	var updatedGroups, skippedGroups, skippedUsers, addedUsers, removedUsers int

	for _, g := range govGroups {
		summary, err := syncGroup(ctx, gc, oc, g)
		if err != nil {
			return err
		}

		// if the summary is nil but there was no error, group was skipped
		if summary == nil {
			skippedGroups++
			continue
		}

		logger.Debug("group membership summary",
			zap.String("governor.group.id", g.ID),
			zap.String("governor.group.slug", g.Slug),
			zap.Any("summary", summary),
		)

		skippedUsers += len(summary.skipped)
		addedUsers += len(summary.added)
		removedUsers += len(summary.removed)

		if len(summary.added) > 0 || len(summary.removed) > 0 {
			updatedGroups++
		}
	}

	logger.Info("completed group membership sync",
		zap.Int("governor.groups.updated", updatedGroups),
		zap.Int("governor.groups.skipped", skippedGroups),
		zap.Int("governor.users.added", addedUsers),
		zap.Int("governor.users.removed", removedUsers),
		zap.Int("governor.users.skipped", skippedUsers),
	)

	return nil
}

func syncGroup(ctx context.Context, gc *governor.Client, oc *okta.Client, g *v1alpha1.Group) (*memberSummary, error) {
	dryRun := viper.GetBool("sync.dryrun")

	l := logger.Desugar().With(
		zap.String("governor.group.id", g.ID),
		zap.String("governor.group.slug", g.Slug),
	)

	// get the details of the governor group (including the membership)
	govGroup, err := gc.Group(ctx, g.ID)
	if err != nil {
		return nil, err
	}

	l.Debug("got governor group details", zap.Any("governor.group", govGroup))

	// get the okta group from the governor id
	oktaGroupID, err := oc.GetGroupByGovernorID(ctx, govGroup.ID)
	if err != nil {
		if errors.Is(err, okta.ErrGroupsNotFound) {
			l.Info("governor group not found in okta, skipping")

			return nil, nil
		}

		l.Error("failed to get okta group by governor id", zap.Error(err))

		return nil, err
	}

	l = l.With(zap.String("okta.group.id", oktaGroupID))

	// get the list of users on the okta group
	oktaGroupMembership, err := oc.ListGroupMembership(ctx, oktaGroupID)
	if err != nil {
		l.Error("failed to list okta group membership", zap.Error(err))
		return nil, err
	}

	l.Debug("got okta group membership", zap.Any("okta.group.members", oktaGroupMembership))

	expectedMembers := []string{}
	skipped := []string{}
	added := []string{}
	removed := []string{}

	for _, member := range oktaGroupMembership {
		user, err := governorUserFromOktaUser(ctx, gc, oc, member, l)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				l.Info("user not found in governor, skipping",
					zap.String("okta.user.id", member.Id),
					zap.Error(err),
				)

				skipped = append(skipped, member.Id)

				continue
			}

			return nil, err
		}

		lg := l.With(
			zap.String("goveror.user.id", user.ID),
			zap.String("goveror.user.email", user.Email),
			zap.String("goveror.user.external_id", user.ExternalID.String),
			zap.String("okta.user.id", member.Id),
		)

		expectedMembers = append(expectedMembers, user.ID)

		if !contains(govGroup.Members, user.ID) {
			lg.Info("adding user to governor group")

			if !dryRun {
				if err := gc.AddGroupMember(ctx, govGroup.ID, user.ID, false); err != nil {
					lg.Error("failed to add group member")
					return nil, err
				}
			}

			added = append(added, member.Id)
		}
	}

	for _, m := range govGroup.Members {
		if !contains(expectedMembers, m) {
			l.Info("pruning user from governor group",
				zap.String("goveror.user.id", m),
			)

			user, err := gc.User(ctx, m, false)
			if err != nil {
				l.Warn("error getting user from governor", zap.String("governor.user.id", m), zap.Error(err))
				continue
			}

			if user.Status.String == "pending" {
				continue
			}

			if !dryRun {
				if err := gc.RemoveGroupMember(ctx, govGroup.ID, m); err != nil {
					l.Error("failed to remove group member",
						zap.String("goveror.user.id", m),
					)

					return nil, err
				}
			}

			removed = append(removed, m)
		}
	}

	return &memberSummary{
		skipped: skipped,
		added:   added,
		removed: removed,
	}, nil
}

func governorUserFromOktaUser(ctx context.Context, gc *governor.Client, oc *okta.Client, oktaUser *okt.User, l *zap.Logger) (*v1alpha1.User, error) {
	email, err := oc.EmailFromUserProfile(oktaUser)
	if err != nil {
		return nil, err
	}

	// get the governor user
	user, ok := userCache[email]
	if !ok {
		u, err := gc.UsersQuery(ctx, map[string][]string{"email": {email}})
		if err != nil {
			return nil, err
		}

		count := len(u)

		switch {
		case count == 0:
			return nil, ErrUserNotFound
		case count > 1:
			return nil, fmt.Errorf("unexpected user count: %d expected 1", count) //nolint:goerr113
		}

		userCache[email] = u[0]
		user = u[0]
	}

	return user, nil
}
