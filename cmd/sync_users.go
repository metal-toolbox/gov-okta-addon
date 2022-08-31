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

// syncUsersCmd governor resources
var syncUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "sync governor and okta resources",
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
	l.Info("starting sync to governor")

	oc, err := okta.NewClient(
		okta.WithLogger(logger.Desugar()),
		okta.WithURL(viper.GetString("okta.url")),
		okta.WithToken(viper.GetString("okta.token")),
		okta.WithCache((!viper.GetBool("okta.nocache"))),
	)
	if err != nil {
		return err
	}

	l.Debug("got okta client", zap.Any("okta client", oc))

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

	l.Debug("got governor client", zap.Any("governor client", gc))

	created, ignored, skipped := 0, 0, 0

	syncFunc := func(ctx context.Context, u *okt.User) (*okt.User, error) {
		l.Debug("processing okta user", zap.String("okta.user.id", u.Id))

		userType, _ := userType(u)
		if userType == "contractor" || userType == "serviceuser" {
			l.Debug("skipping contractor or service user",
				zap.String("okta.user.id", u.Id),
			)

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

		// currently we use pingid here, and this is how external ids live in the gov db
		extID := fmt.Sprintf("ping|%s", externalID)

		// check if user exists in governor
		gUsers, err := gc.UsersQuery(ctx, map[string][]string{"external_id": {extID}})
		if err != nil {
			return nil, err
		}

		l.Debug("got governor users response for external id", zap.Any("governor.users", gUsers))

		if len(gUsers) > 1 {
			logger.Warn("unexpected user count for pingId",
				zap.String("external.id", externalID),
				zap.String("okta.user.id", u.Id),
				zap.Int("response.length", len(gUsers)),
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

		created++

		return u, nil
	}

	l.Info("starting to sync missing okta users into governor")

	users, err := oc.ListUsersWithModifier(ctx, syncFunc, &query.Params{})
	if err != nil {
		return err
	}

	deleted, err := deleteOrphanGovernorUsers(ctx, gc, uniqueExternalIDs(users))
	if err != nil {
		return err
	}

	l.Info("completed user sync",
		zap.Int("created", created),
		zap.Int("deleted", deleted),
		zap.Int("skipped", skipped),
	)

	return nil
}

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

		if extID == "" {
			l.Error("empty external id from okta user",
				zap.Error(err),
				zap.String("okta.user.id", u.Id),
			)

			continue
		}

		e := fmt.Sprintf("ping|%s", extID)

		if _, ok := extIDs[e]; ok {
			l.Warn("external id already exists in list of external ids",
				zap.String("okta.user.id", u.Id),
				zap.String("okta.user.external_id", e),
			)
		}

		extIDs[e] = struct{}{}
	}

	l.Debug("returning list of unique external ids from okta users",
		zap.Int("num.okta.users", len(extIDs)),
	)

	return extIDs
}

func deleteOrphanGovernorUsers(ctx context.Context, gc *governor.Client, extIDMap map[string]struct{}) (int, error) {
	l := logger.Desugar()
	l.Info("starting to clean orphan governor users")

	var deleted int

	govUsers, err := gc.Users(ctx)
	if err != nil {
		return 0, err
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

		if err := gc.DeleteUser(ctx, gu.ID); err != nil {
			return 0, err
		}

		deleted++
	}

	return deleted, nil
}

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

func externalID(u *okt.User) (string, error) {
	l := logger.Desugar()

	// get the external id from the user profile
	for k, v := range *u.Profile {
		if k == "pingSubject" {
			if pv, ok := v.(string); ok {
				return pv, nil
			}

			l.Warn("okta user pingSubject in profile is not a string", zap.String("okta.user.id", u.Id), zap.Any("pingSubject", v))

			return "", ErrOktaUserExternalIDNotString
		}
	}

	return "", fmt.Errorf("external id not found for user %s", u.Id) //nolint:goerr113
}

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
