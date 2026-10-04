module github.com/tpyle/log-manager/logrusmgr/v3

go 1.26.0

require (
	github.com/sirupsen/logrus v1.10.2
	github.com/tpyle/log-manager/v3 v3.0.0
)

require golang.org/x/sys v0.48.0 // indirect

// Development only: replace directives are ignored by modules that depend on
// this one, which use the required version above.
replace github.com/tpyle/log-manager/v3 => ../
