FROM golang:1.23 as builder

# Define build environment variables
ENV GOOS linux
ENV CGO_ENABLED 0

# Add a working directory
WORKDIR /memdb

# Copy files with defined dependencies to the working directory
COPY go.mod go.sum ./

# Download and install dependencies
RUN go mod download

# Copy application files to the working directory
COPY . ./

# Build application and tools
RUN make build


FROM alpine:3.21

# Add a working directory
WORKDIR /memdb

# Copy built binaries and configs
COPY --from=builder /memdb/config.yaml ./
COPY --from=builder /memdb/build/memdb ./
COPY --from=builder /memdb/build/memdb-cli ./

# Define volumes
VOLUME config.yaml

# Expose ports
EXPOSE 7991

# Execute built binary
CMD ./memdb