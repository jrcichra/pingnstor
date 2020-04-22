FROM golang:1.12-alpine3.9 as builder
WORKDIR /app
RUN apk add git g++
COPY . .
RUN go build -v

FROM alpine:3.9
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
