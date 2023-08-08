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
go test -count=1 -v ./...
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

## Testing `swoop caboose argo`

The `argo` version of the `swoop caboose` command requires the full docker
compose environment for testing, with database fixtures loaded. With that
environment running, the service can be started (assuming from the root of the
swoop-go repo):

```shell
go run . caboose argo -f fixtures/swoop-config.yml --kubeconfig=./kubeconfig.yaml
```

However, actually testing `caboose` manually is a rather involved effort due to
the amount of state that needs to be staged in the database, object storage,
and the k8s cluster. To easier facilitate an end-to-end test of `caboose` we
have the bash script `bin/caboose-test.bash`.

That script will:

1. setup a test database, bucket, and k8s namespace
2. build the swoop-go executable
3. create the necessary state in the db/bucket for the test cases
4. run `swoop caboose argo` (runs in the background with a timeout)
5. create the workflow resources in the cluster for the test cases
6. validate everything expected occurred for each test case
7. tear down the test database, bucket, and k8s namespace (unless `CLEANUP !=
   "true"`)

After starting the docker compose environment, run the script like:

```shell
./bin/caboose-test.bash && echo "PASSED tests" || echo "FAILED tests"
```

The exit code of the script should be 0 if everything was as expected, else 1,
hence the echos.

When failures are encountered, it is often helpful to enable "verbose mode"
(make the `VERBOSE` var equal `"true"`) and to disable `CLEANUP` (by setting it
to `"false"`. Note though that with cleanup disabled successive runs of the
script will fail; re-enable cleanup and and run the script once to trigger a
failure and subsequent cleanup before running it again. Also note that deleting
the k8s namespace does take some time, so running the test script in quick
succession will likely fail per the namespace pending deletion.
