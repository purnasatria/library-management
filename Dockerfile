# Start from the official Go image
FROM golang:1.23-alpine AS builder

# Build argument
ARG SERVICE_PATH

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./${SERVICE_PATH}

# Start a new stage from scratch
FROM alpine:3.20.2

RUN apk --no-cache add ca-certificates=20240705-r0

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .
COPY --from=builder /app/api/swagger ./api/swagger
COPY --from=builder /app/migrations ./migrations

# Command to run the executable
CMD ["./main"]
