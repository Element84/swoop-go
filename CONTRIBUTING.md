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
* a Postgres instance with [a SWOOP database](https://github.com/Element84/swoop-db)
* MinIO or some other such S3-compatible object storage service

A docker compose config is provided to simplify running these services, and the
following command will launch three docker containers supplying the serivces
above:

```shell
# load the .env vars
source .env

# bring up the swoop-cabose container in the background
#   --build  forces rebuild of the container in case changes have been made
#   -V       recreates any volumes instead of reusing data
#   -d       run the composed images in daemon mode rather than in the foreground
docker compose up --build -V -d

```

Note that the kubernetes cluster has Argo Workflows custom resource definitions
(CRDs) loaded, but does not have the Argo Workflows server or controller
resources. The aim is to provide an environment that will allow manual or
automated workflow resource creation/updates without having to contend with
Argo Workflows making state changes or attempting to run any workflows.

### Running `swoop-caboose` container

[`./Dockerfile`](./Dockerfile) defines the build steps for a swoop-cabose
container.  To make using the docker container more convenient, the project's
docker-compose will also build and run this container. The repo contents are
installed on `/opt/swoop/` inside the container to help facilitate swoop
operations and testing using the included utilities.

To run the swoop cli after starting the docker compose (detailed above):

```shell
# Run a swoop command interactively on the running swoop-caboose container
docker compose exec swoop-caboose swoop
```

## Manually testing `swoop caboose argo`

The `argo` version of the `swoop caboose` command requires the full docker
compose environment for testing. With that environment running, the service can
be started (assuming from the root of the swoop-go repo):

```shell
go run . caboose argo -f fixtures/swoop-config.yml --kubeconfig=./kubeconfig.yaml
```

Template for simplified workflow resources are included in the `./fixtures`
directory. They can be applied with `sed` and `kubectl`. Here is an example
testing a workflow from the init stage (just created, hasn't been picked up by
the Argo Workflows controller yet), through started and on to successful:

```shell
# create a UUID
UUID="$(uuidgen | tr A-Z a-z)"

# create a workflow resource in the init stage
<./fixtures/resource/init.json sed "s/\${UUID}/${UUID}/" \
    | kubectl --kubeconfig=./kubeconfig.yaml apply -f -

# update the workflow resource to be started
<./fixtures/resource/started.json sed "s/\${UUID}/${UUID}/" \
    | kubectl --kubeconfig=./kubeconfig.yaml apply -f -

# update the workflow resource to be successful
<./fixtures/resource/successful.json sed "s/\${UUID}/${UUID}/" \
    | kubectl --kubeconfig=./kubeconfig.yaml apply -f -
```

Or if you want to test a workflow simply showing up in the failed state:

```shell
UUID="$(uuidgen | tr A-Z a-z)"
<./fixtures/resource/failed.json sed "s/\${UUID}/${UUID}/" \
    | kubectl --kubeconfig=./kubeconfig.yaml apply -f -
```

Testing the start behaivor with caboose is also valuable. Try creating workflow
resources with caboose stopped, then start it and ensure it shows the proper
event handling.
