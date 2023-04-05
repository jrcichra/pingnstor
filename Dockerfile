FROM golang:1.20.3-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bullseye-20230320-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
