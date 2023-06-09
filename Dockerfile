# build swoop-go source and dependencies
FROM golang:1.20-bookworm as APP

WORKDIR /opt/swoop-go

COPY . /opt/swoop-go

# Build swoop
RUN go build -o swoop

FROM golang:1.20-bookworm

ENV SWOOP_DATABASE_HOST=$SWOOP_DATABASE_HOST  \
    SWOOP_DATABASE_PORT=$SWOOP_DATABASE_PORT  \
    SWOOP_DATABASE_USER=$SWOOP_DATABASE_USER  \
    SWOOP_DATABASE_PASSWORD=$SWOOP_DATABASE_PASSWORD  \
    SWOOP_DATABASE_NAME=$SWOOP_DATABASE_NAME  \
    SWOOP_DATABASE_URL_EXTRA=$SWOOP_DATABASE_URL_EXTRA \
    AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
    AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY \
    SWOOP_S3_ENDPOINT=$SWOOP_S3_ENDPOINT \
    SWOOP_BUCKET_NAME=$SWOOP_BUCKET_NAME \
    SWOOP_EXECUTION_DIR=$SWOOP_EXECUTION_DIR

RUN env

WORKDIR /opt/swoop-go

# copy only swoop binary to container image
COPY --from=APP /opt/swoop-go/swoop /opt/swoop-go/

# add binary to image path
ENV PATH=/opt/swoop-go:$PATH
