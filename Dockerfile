# Start with the Alpine go image - lightweight
FROM golang:1.17-alpine
WORKDIR /app
# Copy module and sum list so we can pull modules
COPY go.mod ./
COPY go.sum ./
# Download the modules
RUN go mod download
# Copy any source files to the app directory
COPY *.go ./
# Build the app
RUN go build -o /app/main
# Run the app
CMD ["/app/main"]