package reconciler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystem = "gov_okta_addon"

var (
	groupsCreatedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "groups_created_total",
			Help:      "Total count of groups created.",
		},
	)

	groupsUpdatedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "groups_updated_total",
			Help:      "Total count of groups updated.",
		},
	)

	groupsDeletedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "groups_deleted_total",
			Help:      "Total count of groups deleted.",
		},
	)

	groupsApplicationAssignedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "groups_application_assigned_total",
			Help:      "Total count of applications assigned to groups.",
		},
	)

	groupsApplicationUnassignedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "groups_application_unassigned_total",
			Help:      "Total count of applications unassigned to groups.",
		},
	)

	groupMembershipCreatedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "group_membership_created_total",
			Help:      "Total count of group memberships created.",
		},
	)

	groupMembershipDeletedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "group_membership_deleted_total",
			Help:      "Total count of group memberships deleted.",
		},
	)

	usersDeletedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "users_deleted_total",
			Help:      "Total count of users deleted.",
		},
	)

	usersUpdatedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "users_updated_total",
			Help:      "Total count of users updated.",
		},
	)
)
