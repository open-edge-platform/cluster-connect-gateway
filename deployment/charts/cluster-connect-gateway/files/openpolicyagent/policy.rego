# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

package rbac

default allow := false

allow if {
	have_role
}

role := sprintf("%s_cl-rw", [input.project_id])

have_role if role == input.realm_access.roles[_]
