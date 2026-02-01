FROM golang:1.25.6-bookworm as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
