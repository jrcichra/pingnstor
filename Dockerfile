FROM golang:latest
WORKDIR /
RUN go get "github.com/go-sql-driver/mysql"
RUN go get "github.com/sparrc/go-ping"
COPY pingnstor.go .
RUN go build pingnstor.go
FROM alpine:latest
WORKDIR /
ENV dsn "root@localhost/pingnstor"
ENV f sites.txt
ENV d 60
RUN apk add --no-cache \
    libc6-compat
COPY --from=0 /pingnstor .
CMD ./pingnstor -dsn ${dsn} -f ${f} -d ${d} 