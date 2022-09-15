package cmd

import (
	"context"
	"fmt"
	"net/url"

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

// syncUsersCmd syncs okta users into governor
var syncUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "sync okta users into governor",
	Long: `Performs a one-way user sync from Okta to Governor.
Users that exist in Okta but not in Governor, will be created. Users that exist in Governor but not in Okta, will be deleted.
This command is intended for doing an initial load of users. It is strongly recommended that you use the dry-run flag first 
to see what users would be created/deleted in Governor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return syncUsersToGovernor(cmd.Context())
	},
}

func init() {
	syncCmd.AddCommand(syncUsersCmd)
}

// syncUsersToGovernor syncs users from okta to governor
func syncUsersToGovernor(ctx context.Context) error {
	l := logger.Desugar()

	dryrun := viper.GetBool("sync.dryrun")

	l.Info("starting sync to governor", zap.Bool("dry-run", dryrun))

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
				"read:governor:users",
			},
		}),
	)
	if err != nil {
		return err
	}

	created, ignored, skipped := 0, 0, 0

	// modifier function to get okta users that don't exist in governor and create them
	syncFunc := func(ctx context.Context, u *okt.User) (*okt.User, error) {
		l.Debug("processing okta user", zap.String("okta.user.id", u.Id))

		userType, _ := userType(u)
		if userType == "serviceuser" {
			l.Debug("skipping service user", zap.String("okta.user.id", u.Id))

			ignored++

			return nil, nil
		}

		externalID, err := externalID(u)
		if err != nil {
			return nil, err
		}

		email, err := email(u)
		if err != nil {
			return nil, err
		}

		first, err := firstName(u)
		if err != nil {
			return nil, err
		}

		last, err := lastName(u)
		if err != nil {
			return nil, err
		}

		// the external id in governor is simply the okta id
		extID := externalID

		// check if user exists in governor
		gUsers, err := gc.UsersQuery(ctx, map[string][]string{"external_id": {extID}})
		if err != nil {
			return nil, err
		}

		l.Debug("got governor users response for external id", zap.Any("governor.users", gUsers))

		if len(gUsers) > 1 {
			logger.Warn("unexpected user count for external_id",
				zap.String("external.id", externalID),
				zap.String("okta.user.id", u.Id),
				zap.Int("num.governor.users", len(gUsers)),
			)

			return nil, nil
		}

		if len(gUsers) == 1 {
			logger.Debug("user exists in governor",
				zap.String("external.id", externalID),
				zap.String("okta.user.id", u.Id),
				zap.String("governor.user.id", gUsers[0].ID),
			)

			return u, nil
		}

		l.Info("user not found in governor, creating",
			zap.String("external.id", externalID),
			zap.String("okta.user.id", u.Id),
			zap.String("okta.user.email", email),
		)

		if !dryrun {
			gUser, err := gc.CreateUser(ctx, &v1alpha1.UserReq{
				Email:      email,
				ExternalID: extID,
				Name:       fmt.Sprintf("%s %s", first, last),
			})
			if err != nil {
				return nil, err
			}

			l.Debug("created governor user from okta sync",
				zap.String("governor.user.id", gUser.ID),
				zap.String("external.id", externalID),
				zap.String("okta.user.id", u.Id),
				zap.String("okta.user.email", email),
			)
		}

		created++

		return u, nil
	}

	l.Info("starting to sync missing okta users into governor", zap.Bool("dry-run", dryrun))

	users, err := oc.ListUsersWithModifier(ctx, syncFunc, &query.Params{})
	if err != nil {
		return err
	}

	deleted, err := deleteOrphanGovernorUsers(ctx, gc, uniqueExternalIDs(users))
	if err != nil {
		return err
	}

	l.Info("completed user sync",
		zap.Int("governor.users.created", created),
		zap.Int("governor.users.deleted", deleted),
		zap.Int("governor.users.skipped", skipped),
	)

	return nil
}

// deleteOrphanGovernorUsers is a helper function to delete governor users that not longer exist in okta
func deleteOrphanGovernorUsers(ctx context.Context, gc *governor.Client, extIDMap map[string]struct{}) (int, error) {
	dryrun := viper.GetBool("sync.dryrun")

	l := logger.Desugar()
	l.Info("starting to clean orphan governor users", zap.Bool("dry-run", dryrun))

	var deleted int

	govUsers, err := gc.Users(ctx, false)
	if err != nil {
		return deleted, err
	}

	l.Debug("got list of governor users to compare to okta users",
		zap.Int("num.governor.users", len(govUsers)),
		zap.Int("num.okta.users", len(extIDMap)),
	)

	for _, gu := range govUsers {
		if gu.ExternalID == "" {
			l.Warn("governor user is missing external id, won't delete",
				zap.String("governor.user.id", gu.ID),
				zap.String("governor.user.email", gu.Email),
			)

			continue
		}

		if _, ok := extIDMap[gu.ExternalID]; ok {
			l.Debug("governor user exists in okta, continuing",
				zap.String("governor.user.id", gu.ID),
				zap.String("governor.user.external_id", gu.ExternalID),
				zap.String("governor.user.email", gu.Email),
			)

			continue
		}

		l.Info("governor user doesn't exist in okta, deleting",
			zap.String("governor.user.id", gu.ID),
			zap.String("governor.user.external_id", gu.ExternalID),
			zap.String("governor.user.email", gu.Email),
		)

		if !dryrun {
			if err := gc.DeleteUser(ctx, gu.ID); err != nil {
				return deleted, err
			}
		}

		deleted++
	}

	return deleted, nil
}

// uniqueExternalIDs builds a map of unique external ids from a list of okta users
func uniqueExternalIDs(users []*okt.User) map[string]struct{} {
	l := logger.Desugar()

	l.Debug("generating list of unique external ids from okta users",
		zap.Int("num.okta.users", len(users)),
	)

	extIDs := map[string]struct{}{}

	for _, u := range users {
		extID, err := externalID(u)
		if err != nil {
			l.Error("error getting external id from okta user",
				zap.Error(err),
				zap.String("okta.user.id", u.Id),
			)

			continue
		}

		if _, ok := extIDs[extID]; ok {
			l.Warn("external id already exists in list of external ids",
				zap.String("okta.user.id", u.Id),
				zap.String("okta.user.external_id", extID),
			)
		}

		extIDs[extID] = struct{}{}
	}

	l.Debug("returning list of unique external ids from okta users",
		zap.Int("num.okta.users", len(extIDs)),
	)

	return extIDs
}

// email parses the email from the okta user profile
func email(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the email from the user profile
	for k, v := range *u.Profile {
		if k == "email" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			l.Warn("okta user email in profile is not a string", zap.String("okta.user.id", u.Id), zap.Any("okta.user.email", v))

			return "", ErrOktaUserEmailNotString
		}
	}

	return "", fmt.Errorf("email not found for user %s", u.Id) //nolint:goerr113
}

// externalID returns the id for the okta user
func externalID(u *okt.User) (string, error) {
	if u.Id == "" {
		return "", ErrOktaUserIDEmpty
	}

	return u.Id, nil
}

// userType parses the userType from the okta user profile
func userType(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the userType from the user profile
	for k, v := range *u.Profile {
		if k == "userType" {
			if pv, ok := v.(string); ok {
				return pv, nil
			}

			l.Warn("okta user userType in profile is not a string", zap.String("okta.user.id", u.Id), zap.Any("userType", v))

			return "", ErrOktaUserTypeNotString
		}
	}

	return "", fmt.Errorf("userType not found for user %s", u.Id) //nolint:goerr113
}

// firstName parses the firstName from the okta user profile
func firstName(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the firstName from the user profile
	for k, v := range *u.Profile {
		if k == "firstName" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			l.Warn("okta user first name in profile is not a string", zap.String("okta.user.id", u.Id), zap.Any("okta.user.email", v))

			return "", ErrOktaUserFirstNameNotString
		}
	}

	return "", fmt.Errorf("firstName not found for user %s", u.Id) //nolint:goerr113
}

// lastName parses the lastName from the okta user profile
func lastName(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the lastName from the user profile
	for k, v := range *u.Profile {
		if k == "lastName" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			l.Warn("okta user last name in profile is not a string", zap.String("okta.user.id", u.Id), zap.Any("okta.user.email", v))

			return "", ErrOktaUserLastNameNotString
		}
	}

	return "", fmt.Errorf("lastName not found for user %s", u.Id) //nolint:goerr113
}
