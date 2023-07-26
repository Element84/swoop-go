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

# add binary to image path
ENV PATH=/opt/swoop-go:$PATH

ENTRYPOINT [ "swoop caboose argo" ]
