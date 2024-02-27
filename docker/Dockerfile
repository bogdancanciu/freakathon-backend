# Use the official Go image as a base
FROM golang:1.21 AS builder

ENV GO111MODULE=on

# Set the working directory inside the container
WORKDIR /freakathon

# Copy the Go mod and sum files
COPY go.mod ./
COPY go.sum ./

# Download dependencies only if the mod/sum files have changed
RUN go mod download

# Copy the entire project directory into the container
COPY . .

# Build the Go app
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o backend ./cmd/main.go
# Start a new stage from scratch
FROM debian:latest

# Set the working directory inside the container
WORKDIR /freakathon-backend

# Copy the compiled binary from the builder stage
COPY --from=builder /freakathon/backend .

# Expose port 8080 to the outside world
EXPOSE 8090

# Command to run the executable
CMD ["./backend", "serve"]
