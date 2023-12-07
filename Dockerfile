FROM golang:1.21.5-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bullseye-20231120-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
