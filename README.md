# pingnstor [![Build Status](https://travis-ci.com/jrcichra/pingnstor.svg?branch=master)](https://travis-ci.com/jrcichra/pingnstor)

Simple Go program that stores ping data in a database

## Config

+ Look at config.yml for an example config file
+ Look at setup.sql for the expected table structure
+ You'll probably need libc & root privs to send ICMP packets

## Help

```bash
./pingnstor-windows-amd64.exe -h
Usage of pingnstor-windows-amd64.exe:
  -dsn string
        The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name
  -f string
        Newline separated list of domains to ping (default is "config.yml")
```
