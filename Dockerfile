FROM golang:latest
WORKDIR /
RUN go get "github.com/go-sql-driver/mysql"
RUN go get "github.com/sparrc/go-ping"
COPY pingnstor.go .
RUN go build pingnstor.go
FROM alpine:latest
WORKDIR /
ENV dsn /
ENV f sites.txt
ENV d 60
COPY --from=0 /pingnstor .
CMD ./pingnstor -dsn ${dsn} -f ${f} -d ${d} 