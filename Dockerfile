FROM golang:1.24.4-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bullseye-20250520-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
