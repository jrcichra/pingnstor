# pingnstor ![Actions Status](https://github.com/jrcichra/pingnstor/workflows/pingnstor/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/jrcichra/pingnstor)](https://goreportcard.com/report/github.com/jrcichra/pingnstor)

Simple Go program that stores ping data in a database

## Docker

```bash
docker run --name=pingnstor --volume=/home/pi/pingnstor/config.yml:/config.yml \
--privileged --restart=unless-stopped -d --network=host -t \
ghcr.io/jrcichra/pingnstor -f /config.yml -dsn 'pingnstor:test@tcp(mariadb)/pingnstor'
```

## Config

- Look at config.yml for an example config file
- Look at setup.sql for the expected table structure
- You'll probably need libc & root privs to send ICMP packets

## Help

```bash
Usage of ./pingnstor:
  -dbtype string
        database to connect to (default "mysql")
  -dnsRefresh int
        minutes between dns refreshes (default 15)
  -dsn string
        The connection string for your database
  -f string
        YAML configuration file (default "config.yml")
  -listen string
        http metrics/debug server listen address (default ":9103")
```

# DSNs

postgres docs: https://pkg.go.dev/github.com/lib/pq@v1.10.7?utm_source=gopls#hdr-Connection_String_Parameters

mysql docs: https://github.com/go-sql-driver/mysql#dsn-data-source-name
