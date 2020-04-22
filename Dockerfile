FROM golang:1.13-alpine3.10 as builder
WORKDIR /app
RUN apk add git g++
COPY . .
RUN go build -v

FROM alpine:3.10
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
