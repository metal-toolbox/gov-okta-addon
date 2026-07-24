package cmd

import (
	"context"
	"fmt"

	"github.com/metal-toolbox/gov-okta-addon/internal/configs"
	"github.com/metal-toolbox/gov-okta-addon/internal/okta"
	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	governor "github.com/metal-toolbox/governor-api/pkg/client"
	okt "github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// syncUsersCmd syncs okta users into governor
var syncUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "sync okta users into governor",
	Long: `Performs a one-way user sync from Okta to Governor.
Users that exist in Okta but not in Governor, will be created. Users that exist in Governor but not in Okta, will be deleted.
This command is intended for doing an initial load of users. It is strongly recommended that you use the dry-run flag first 
to see what users would be created/deleted in Governor.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return syncUsersToGovernor(cmd.Context())
	},
}

func init() {
	syncCmd.AddCommand(syncUsersCmd)
}

// syncUsersToGovernor syncs users from okta to governor
func syncUsersToGovernor(ctx context.Context) error {
	logger := logger.Desugar()
	dryRun := configs.AppConfig.DryRun

	logger.Info("starting sync to governor users", zap.Bool("dry-run", dryRun))

	oc, err := configs.NewOktaClient(logger)
	if err != nil {
		return err
	}

	gc, err := configs.NewGovernorClient(ctx, governor.WithLogger(logger))
	if err != nil {
		return err
	}

	created, skipped, updated := 0, 0, 0

	// modifier function to get okta users that don't exist in governor and create them
	syncFunc := func(ctx context.Context, u *okt.User) (*okt.User, error) {
		logger.Debug("processing okta user", zap.String("okta.user.id", u.GetId()))

		email, err := okta.EmailFromUserProfile(u)
		if err != nil {
			return nil, err
		}

		first, err := okta.FirstNameFromUserProfile(u)
		if err != nil {
			return nil, err
		}

		last, err := okta.LastNameFromUserProfile(u)
		if err != nil {
			return nil, err
		}

		// the external id in governor is simply the okta id
		extID := u.GetId()

		// check if user exists in governor
		gUsers, err := gc.UsersQuery(ctx, map[string][]string{"email": {email}})
		if err != nil {
			return nil, err
		}

		logger.Debug("got governor users response for email", zap.Any("governor.users", gUsers))

		if len(gUsers) > 1 {
			logger.Warn("unexpected user count for email",
				zap.String("okta.user.email", email),
				zap.String("okta.user.id", u.GetId()),
				zap.Int("num.governor.users", len(gUsers)),
			)

			return nil, nil
		}

		if len(gUsers) == 1 {
			gUser := gUsers[0]

			l := logger.With(zap.String("okta.user.email", email),
				zap.String("okta.user.id", u.GetId()),
				zap.String("governor.user.id", gUser.ID))

			if gUser.Status.String != v1alpha1.UserStatusPending {
				l.Debug("user exists in governor and is not pending")
				return u, nil
			}

			logger.Info("user exists in governor and is marked pending, marking active",
				zap.String("okta.user.id", u.GetId()),
				zap.String("okta.user.email", email),
			)

			if !dryRun {
				gUser, err := gc.UpdateUser(ctx, gUser.ID,
					&v1alpha1.UserReq{
						Email:      email,
						ExternalID: extID,
						Name:       fmt.Sprintf("%s %s", first, last),
						Status:     v1alpha1.UserStatusActive,
					})
				if err != nil {
					return nil, err
				}

				l.Debug("updated governor user from okta sync",
					zap.String("governor.user.id", gUser.ID),
					zap.String("okta.user.id", u.GetId()),
					zap.String("okta.user.email", email),
				)
			}

			updated++

			return u, nil
		}

		logger.Info("user not found in governor, creating",
			zap.String("okta.user.id", u.GetId()),
			zap.String("okta.user.email", email),
		)

		if !dryRun {
			gUser, err := gc.CreateUser(ctx, &v1alpha1.UserReq{
				Email:      email,
				ExternalID: extID,
				Name:       fmt.Sprintf("%s %s", first, last),
				Status:     v1alpha1.UserStatusActive,
			})
			if err != nil {
				return nil, err
			}

			logger.Debug("created governor user from okta sync",
				zap.String("governor.user.id", gUser.ID),
				zap.String("okta.user.id", u.GetId()),
				zap.String("okta.user.email", email),
			)
		}

		created++

		return u, nil
	}

	logger.Info("starting to sync missing okta users into governor", zap.Bool("dry-run", dryRun))

	users, err := oc.ListUsersWithModifier(ctx, syncFunc, "")
	if err != nil {
		return err
	}

	deleted, err := deleteOrphanGovernorUsers(ctx, gc, uniqueEmails(users))
	if err != nil {
		return err
	}

	logger.Info("completed user sync",
		zap.Int("governor.users.created", created),
		zap.Int("governor.users.deleted", deleted),
		zap.Int("governor.users.skipped", skipped),
		zap.Int("governor.users.updated", updated),
	)

	return nil
}

// deleteOrphanGovernorUsers is a helper function to delete governor users that not longer exist in okta
func deleteOrphanGovernorUsers(ctx context.Context, gc *governor.Client, emailIDMap map[string]string) (int, error) {
	dryRun := configs.AppConfig.DryRun
	l := logger.Desugar()

	l.Info("starting to clean orphan governor users", zap.Bool("dry-run", dryRun))

	var deleted int

	govUsers, err := gc.UsersV2(ctx, map[string][]string{})
	if err != nil {
		return deleted, err
	}

	l.Debug("got list of governor users to compare to okta users",
		zap.Int("num.governor.users", len(govUsers)),
		zap.Int("num.okta.users", len(emailIDMap)),
	)

	for _, gu := range govUsers {
		if gu.Status.String == v1alpha1.UserStatusPending {
			l.Debug("skipping pending governor user",
				zap.String("governor.user.id", gu.ID),
				zap.String("governor.user.email", gu.Email))

			continue
		}

		govEmail := gu.Email
		if govEmail == "" {
			l.Warn("governor user is missing email, won't delete",
				zap.String("governor.user.id", gu.ID),
				zap.String("governor.user.email", gu.Email),
			)

			continue
		}

		if id, ok := emailIDMap[govEmail]; ok {
			l.Debug("governor user exists in okta, continuing",
				zap.String("governor.user.id", gu.ID),
				zap.String("okta.user.id", id),
				zap.String("governor.user.email", gu.Email),
			)

			continue
		}

		l.Info("governor user doesn't exist in okta, deleting",
			zap.String("governor.user.id", gu.ID),
			zap.String("governor.user.email", gu.Email),
		)

		if !dryRun {
			if err := gc.DeleteUser(ctx, gu.ID); err != nil {
				return deleted, err
			}
		}

		deleted++
	}

	return deleted, nil
}

// uniqueExternalIDs builds a map of unique emails from a list of okta users
func uniqueEmails(users []*okt.User) map[string]string {
	l := logger.Desugar()

	l.Debug("generating list of unique emails from okta users",
		zap.Int("num.okta.users", len(users)),
	)

	emails := map[string]string{}

	for _, u := range users {
		email, err := okta.EmailFromUserProfile(u)
		if err != nil {
			l.Error("error getting email address from okta user",
				zap.Error(err),
				zap.String("okta.user.id", u.GetId()),
			)

			continue
		}

		if _, ok := emails[email]; ok {
			l.Info("email already exists in list of emails",
				zap.String("okta.user.id", u.GetId()),
				zap.String("okta.user.email", email),
			)
		}

		emails[email] = u.GetId()
	}

	l.Debug("returning list of unique email address from okta users",
		zap.Int("num.okta.users", len(emails)),
	)

	return emails
}

// userType parses the userType from the okta user profile
func userType(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the userType from the user profile
	if u.Profile != nil && u.Profile.UserType.Get() != nil {
		return *u.Profile.UserType.Get(), nil
	}

	l.Debug("userType not found in okta user profile", zap.String("okta.user.id", u.GetId()))

	return "", fmt.Errorf("userType not found for user %s", u.GetId()) //nolint:err113
}
