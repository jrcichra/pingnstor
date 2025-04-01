FROM golang:1.24.1-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bullseye-20250224-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
