# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package rbac

default allow := false

allow if {
	have_role
}

role := sprintf("%s_cl-rw", [input.project_id])

have_role if role == input.realm_access.roles[_]

allow if service_group_access

service_group_access if {
	"apps-m2m-service-account" in input.groups
	"clusters-read-role" in input.realm_access.roles
	input.preferred_username == "service-account-co-manager-m2m-client"
}
