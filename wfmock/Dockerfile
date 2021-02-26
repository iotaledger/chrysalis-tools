############################
# Build
############################
FROM golang:latest as build-env

# Set the current working directory inside the container
RUN mkdir /wfmock
WORKDIR /wfmock

# Use Go Modules
COPY go.mod .
COPY go.sum .

ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Build the binary
RUN go build -ldflags='-w -s' -o /app/wfmock

############################
# Image
############################
FROM gcr.io/distroless/base

EXPOSE 14266/tcp

# Copy the binary into /app with write access
COPY --from=build-env --chown=nonroot:nonroot /app /app
WORKDIR /app

# Copy the configuration
COPY config.json config.json

CMD ["/app/wfmock"]
