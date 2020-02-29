# pingnstor ![Actions Status](https://github.com/jrcichra/pingnstor/workflows/pingnstor/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/jrcichra/pingnstor)](https://goreportcard.com/report/github.com/jrcichra/pingnstor)

Simple Go program that stores ping data in a database

## Docker
```bash
docker run --name=pingnstor --volume=/home/pi/pingnstor/config.yml:/config.yml --privileged --expose=8080 --restart=always --detach=true --network=host -t jrcichra/pingnstor -f /config.yml -hopint 5 -hopnum 4 -dsn 'pingnstor:test@tcp(mariadb)/pingnstor'
```

## Config

+ Look at config.yml for an example config file
+ Look at setup.sql for the expected table structure
+ You'll probably need libc & root privs to send ICMP packets

## Help

```bash
Usage of ./pingnstor:
  -dsn string
        The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name
  -f string
        YAML configuration file (default "config.yml")
  -hopint int
        Automatically determine the next hop on startup and use this interval
  -hopnum int
        Number of the hop you want to ping
```
