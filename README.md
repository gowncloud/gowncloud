# gowncloud: Proof of concept

Goal: implement a working proof of concept with a backend written in golang.

## install:

```
go get github.com/gowncloud/gowncloud
cd $GOPATH/src/github.com/gowncloud/gowncloud
go generate
go build
./gowncloud -c [client_id] -s [client_secret] --db "[database_driver]://[database_user]@[database_url]:[database_port]?sslmode=disable"

The client_id and client_secret should come from an organization api key in itsyou.online
```

Currently the only supported database driver is `postgres`. sslmode can be changed
to require if a secure database is available, however this is discouraged for testing purposes.
An example connection string is:

`"postgres://root@localhost:26257?sslmode=disable"`

The only supported database is [cockroachdb](https://github.com/cockroachdb/cockroach).

navigate to `localhost:8080/index.php`

## Optional parameters:

`--dav-directory`: allows the user to specify the path to the root directory of the webdav server.
All files uploaded to gowncloud will be stored here. The directory (tree) will be created
by gowncloud as required. To avoid conflicts, the path should point to an unexisting
or completely empty directory. Synonym: `--dir`
