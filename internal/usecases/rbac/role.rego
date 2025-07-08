package rbac

import rego.v1

# There are 4 roles in the system
# root
# admin
# user

resources := {}

default allow := false

allow if {
	input.role == "user_role_root"
}

allow if {
	input.role in resources[input.path][input.method]
}
