# pingnstor [![Build Status](https://travis-ci.org/jrcichra/pingnstor.svg?branch=master)](https://travis-ci.org/jrcichra/pingnstor)

Simple Go program that stores ping data in a database

```bash
./pingnstor-windows-amd64.exe -h
Usage of pingnstor-windows-amd64.exe:
  -d int
        delay (in seconds) between all pings (pings happen independently of each other and will go out of sync) (default 60)
  -dsn string
        The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name
  -f string
        Newline separated list of domains to ping (default "sites.txt")
```

You'll probably need libc & root privs to send ICMP packets.

Look at setup.sql for the expected table structure.
