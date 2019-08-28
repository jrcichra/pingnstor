package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sparrc/go-ping"
)

type pResp struct {
	domain string
	rtt    time.Duration
}

func p(site string, c chan pResp, delay int64) {
	done := false
	for !done {
		pinger, err := ping.NewPinger(site)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			done = true
			return
		}
		pinger.SetPrivileged(true)

		pinger.OnRecv = func(pkt *ping.Packet) {

		}
		pinger.OnFinish = func(stats *ping.Statistics) {
			c <- pResp{domain: site, rtt: stats.MaxRtt}
		}

		//fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
		pinger.Count = 1
		pinger.Run()
		fmt.Println("I am pinging", site, "and should be delaying for", delay, "seconds")
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func main() {
	//get flags
	dsn := flag.String("dsn", "", "The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name")
	filename := flag.String("f", "sites.txt", "Newline separated list of domains to ping")
	delay := flag.Int64("d", 60, "delay (in seconds) between all pings (pings happen independently of each other and will go out of sync)")

	flag.Parse()

	// connect to the database
	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Read the file with all the urls you want to ping...
	file, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)

	//channel that all p's will send back to the main thread on
	c := make(chan pResp)
	for scanner.Scan() {
		//spawn a pinger with a delay for this
		go p(scanner.Text(), c, *delay)
	}
	//loop through every response and process the input for the DB
	for {
		//block on the channel?
		r := <-c
		// prepare the query
		stmt, err := db.Prepare("insert pings set domain = ?, packet_rtt = ?")
		if err != nil {
			log.Fatal(err)
		}
		res, err := stmt.Exec(r.domain, r.rtt.Seconds())
		if err != nil {
			log.Fatal(err)
		}
		_, err = res.RowsAffected()
		if err != nil {
			log.Fatal(err)
		}

	}
}
