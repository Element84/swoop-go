# Instructions for development and contributions

This project requires `go`. See [`go.mod`](go.mod) for the current required
`go` version.

## pre-commit hooks

Several pre-commit hooks are configured in
[`.pre-commit-config.yaml`](.pre-commit-config.yaml).  Ensure the
[`pre-commit`](https://pre-commit.com/) tool is available in your environment,
then install the pre-commit git hook by running `pre-commit install`.

## Common `go` commands

### Building the `swoop` binary

```shell
# from the project root
go build -o swoop
```

### Running tests

```shell
# from the project root
go test
```

### Managing dependencies

Simply add/remove imports as needed in modules.  Use the following commands to
pull in/clean up installed modules.

```shell
# from the project root

# update go.mod from src imports
go mod tidy

# upgrade all dependencies
go get -u

# upgrade and clean up
go get -u && go mod tidy
```

### Formatting `.go` files

```shell
# from the project root
gofmt -s -w -l .
```

Note that `gofmt` enforces the standard of tabs for indentation and spaces for
alignment, and this runs as a pre-commit hook. Best practice is to ensure your
editor is configured to use use tabs with your preferred tabstop width.

## Running `swoop`

After building the `swoop` binary, it can be executed via `./swoop`.

Note that some of the exposed commands may require one or more external
services for testing, including but not limited to:

* a kubernetes cluster running Argo Workflows
* a Postgres instance with [a SWOOP database](https://github.com/Element84/swoop/tree/main/db)
* MinIO or some other such S3-compatible object storage service

### Running `swoop-caboose` container

[`./Dockerfile`](./Dockerfile) defines the build steps for a swoop-cabose
container.  To make using the docker container more convenient, a `docker-compose.yml`
file is provided in the project root. The repo contents are installed on `/opt/swoop/`
inside the container to help facilitate swoop operations and testing using
the included utilities. For example, to bring up swoop:

```shell
# load the .env vars
source .env

# bring up the swoop-cabose container in the background
#   --build  forces rebuild of the container in case changes have been made
#   -V       recreates any volumes instead of reusing data
#   -d       run the composed images in daemon mode rather than in the foreground
docker compose up --build -V -d

# Run a swoop command interactively on the running swoop-caboose container
docker compose exec swoop-caboose swoop
```
