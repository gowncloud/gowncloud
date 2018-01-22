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

## Optional parameters:

`--dav-directory`: allows the user to specify the path to the root directory of the webdav server.
All files uploaded to gowncloud will be stored here. The directory (tree) will be created
by gowncloud as required. To avoid conflicts, the path should point to an unexisting
or completely empty directory. Synonym: `--dir`

## Authentication

### Interactive session
When accessing gowncloud through a browser, a normal OAuth2 flow is used to authenticate. Gowncloud gets a jwt from itsyou.online and stores this in a sessioncookie.

### API access
It is possible to access the webdav url's and other endpoints by providing a jwt yourself. The OAuth2 `client_id` must be in the audience ('aud' claim) list.
There are 2 ways to supply the jwt:
1. Via the `Authorization` header (example: `Authorization: bearer JWT`)
2. By using basic authentication, the user is ignored but the jwt should be passed as a password.
