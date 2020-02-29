FROM golang:alpine3.11 as builder
WORKDIR /app
RUN apk add git g++
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

FROM alpine:3.11
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
