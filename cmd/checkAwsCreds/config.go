package main

import (
	"flag"
)

type structFlags struct {
	AccessKey *string
	SecretKey *string
	Token     *string
	Region    *string
}

// Flags is a struct for command line flags ready to be utilized by flag.Parse()
var Flags = structFlags{
	AccessKey: flag.String("AccessKey", "", "AccessKey AWS credential"),
	SecretKey: flag.String("SecretKey", "", "SecretKey AWS credential"),
	Token:     flag.String("Token", "", "Token AWS credential"),
	Region:    flag.String("Region", "", "Region AWS credential"),
}

// ParseFlags is a dummy flag.Parse() wrapper
func ParseFlags() {
	flag.Parse()
}
