module example/adapters-sql

go 1.25.0

require (
	github.com/lib/pq v1.10.9
	github.com/ragokan/limen v0.1.8
	github.com/ragokan/limen/adapters/sql v0.1.8
	github.com/ragokan/limen/plugins/credential-password v0.1.8
)

require (
	github.com/jmoiron/sqlx v1.4.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/ragokan/limen => ../../..

replace github.com/ragokan/limen/adapters/sql => ../../../adapters/sql

replace github.com/ragokan/limen/plugins/credential-password => ../../../plugins/credential-password
