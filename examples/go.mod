module github.com/tpyle/log-manager/examples

go 1.26.0

require (
	github.com/rs/zerolog v1.35.1
	github.com/sirupsen/logrus v1.10.2
	github.com/tpyle/log-manager/logrusmgr/v3 v3.0.0
	github.com/tpyle/log-manager/v3 v3.0.0
	github.com/tpyle/log-manager/zerologmgr/v3 v3.0.0
)

require (
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

// The examples always build against the code in this repository.
replace (
	github.com/tpyle/log-manager/logrusmgr/v3 => ../logrusmgr
	github.com/tpyle/log-manager/v3 => ../
	github.com/tpyle/log-manager/zerologmgr/v3 => ../zerologmgr
)
