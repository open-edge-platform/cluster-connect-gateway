# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package rbac

default allow := false

allow if {
	have_role
}

role := sprintf("%s_cl-rw", [input.project_id])

have_role if role == input.realm_access.roles[_]

# allow certain m2m groups limited cluster access
# it does NOT grant write; write still requires the concrete <project>_cl-rw role handled by have_role
allow if {
	service_group_access
}

# set of groups whose membership should allow read-level cluster interactions
service_groups := {"apps-m2m-service-account", "edge-manager-group"}

service_group_member if {
	service_groups[input.groups[_]]
}

has_cluster_read_role if {
	input.realm_access.roles[_] == "clusters-read-role"
}

project_scoped if {
	input.project_id != ""
}

service_group_access if {
	service_group_member
	has_cluster_read_role
	project_scoped
}
