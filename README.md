# gowncloud: Proof of concept

Goal: implement a working proof of concept with a backend written in golang.

## install:

```
go build
./gowncloud -c [client_id] -s [client_secret] --db "[database_driver]://[database_user]@[database_url]:[database_port]?sslmode=disable"

The client_id and client_secret should come from an organization api key in itsyou.online
```

Currently the only supported database driver is `postgres`. sslmode can be changed
to require if a secure database is available, however this is discouraged for testing purposes.
An example connection string is:

`"postgres://root@localhost:26257?sslmode=disable"`

Development is done with [cockroachdb](https://github.com/cockroachdb/cockroach).
This database uses the postgres driver, making them freely interchangeable.

navigate to `localhost:8080/index.php`
