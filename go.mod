module github.com/linxlib/fw_openapi

go 1.22.0

replace (
	github.com/linxlib/astp => ../astp
	github.com/linxlib/fw => ../../repos/fw
)
require (
	github.com/sv-tools/openapi v0.4.0
	github.com/linxlib/fw v0.0.0-00010101000000-000000000000
	github.com/linxlib/astp v0.0.0-00010101000000-000000000000
)