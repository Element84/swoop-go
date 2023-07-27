# build swoop-go source and dependencies
FROM golang:1.20-bookworm as APP

WORKDIR /opt/swoop-go

COPY . /opt/swoop-go

# Build swoop
RUN go build -o swoop

FROM golang:1.20-bookworm

WORKDIR /opt/swoop-go

# copy only swoop binary to container image
COPY --from=APP /opt/swoop-go/swoop /opt/swoop-go/

# Temporary solution of adding the fixtures directory to the image.
# This is in place to allow the swoop caboose container to find a
# version of the swoop-config.yml. Once we centralize an unified
# swoop-config file that's accessible and compatible across swoop services
# this line to copy the fixtures into the swoop-go image should be removed.
COPY --from=APP /opt/swoop-go/fixtures /opt/swoop-go/fixtures

# add binary to image path
ENV PATH=/opt/swoop-go:$PATH
