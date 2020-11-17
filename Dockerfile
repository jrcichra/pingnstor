FROM golang:1.15-buster as builder
WORKDIR /app
RUN apt-get update && apt-get install -y git g++
COPY . .
RUN go build -v

FROM debian:buster-20201012-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
