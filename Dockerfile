FROM golang:1.19.5-bullseye as builder
WORKDIR /app
COPY . .
RUN go build -v

FROM debian:bullseye-20221219-slim
WORKDIR /app
COPY --from=builder /app/pingnstor .

# Command to run the executable
ENTRYPOINT ["/app/pingnstor"] 
