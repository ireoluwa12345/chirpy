package main

import (
	_ "github.com/lib/pq"
)

const (
	accessTokenExpiry string = "86400s"
	// refreshTokenExpiry is created in hours
	refreshTokenExpiry int = 60 * 24
)
