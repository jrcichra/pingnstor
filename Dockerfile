FROM golang:1.13-buster as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:buster-20201012-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
