package workspace

import "errors"

var errOrgRequired = errors.New("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
