FROM golang:1.20-bookworm

WORKDIR /opt/swoop-go


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

COPY . /opt/swoop-go

# Build swoop
RUN go build -o swoop

ENV PATH=/opt/swoop-go:$PATH
