package main

import (
	"log"
)

const (
	unset     = "unset"
	spaghetti = "spaghetti"
)

var ( // build info
	version   = unset
	date      = unset
	commit    = unset
	appname   = spaghetti
	goversion = unset
)

func printVersion() {
	log.Printf("app_name: %s\n"+
		"version: %s\n"+
		"build_time: %s\n"+
		"commit: %s\n"+
		"go_version: %s\n", appname, version, date, commit, goversion)
}
