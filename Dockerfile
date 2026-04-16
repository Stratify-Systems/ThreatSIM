# Build stage
FROM golang:alpine AS builder

# Set the working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Compile the Go application statically
RUN CGO_ENABLED=0 GOOS=linux go build -o threatsim ./cmd/threatsim

# Runtime stage
FROM alpine:latest

# Add certificates and timezone data for secure connections and logging
RUN apk --no-cache add ca-certificates tzdata

# Set the working directory
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/threatsim .

# Copy directories required for the app to function properly
COPY --from=builder /app/configs/ ./configs/
COPY --from=builder /app/db/ ./db/

# Expose the API port
EXPOSE 8080

# Command to run the application
CMD ["./threatsim", "server"]
