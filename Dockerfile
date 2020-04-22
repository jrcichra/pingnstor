FROM golang:1.14.2-alpine3.11 as builder
WORKDIR /app
RUN apk add git g++
COPY . .
RUN go build -v

FROM alpine:3.11
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
