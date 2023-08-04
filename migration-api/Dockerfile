############################
# Build
############################
FROM golang:1.20-bullseye AS build

# Set the current working directory inside the container
RUN mkdir /migration-api
WORKDIR /migration-api

# Use Go Modules
COPY go.mod .
COPY go.sum .

RUN go mod download
RUN go mod verify

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Build the binary
RUN go build -ldflags='-w -s' -o /app/migration-api

############################
# Image
############################
FROM gcr.io/distroless/cc-debian11:nonroot

EXPOSE 8484/tcp

# Copy the binary into /app with write access
COPY --from=build --chown=nonroot:nonroot /app /app
WORKDIR /app

# Copy the configuration
COPY config.json config.json

CMD ["/app/migration-api"]