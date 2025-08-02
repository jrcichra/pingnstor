FROM golang:1.24.5-bookworm as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
