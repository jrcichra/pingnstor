package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"net/http"
	_ "net/http/pprof"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/oklog/run"
	"gopkg.in/yaml.v3"
)

type pResp struct {
	domain    string
	ipAddress string
	rtt       time.Duration
}

type ConfigDomain struct {
	Delay int `yaml:"delay"`
}

type Config struct {
	Domains map[string]ConfigDomain `yaml:"domains"`
}

func p(ctx context.Context, dbChan chan pResp, domain string, ipAddress string) error {
	// initialize a pinger
	pinger, err := ping.NewPinger(ipAddress)
	if err != nil {
		return err
	}
	pinger.SetPrivileged(true)
	pinger.OnRecv = func(pkt *ping.Packet) {}
	pinger.OnFinish = func(stats *ping.Statistics) {
		dbChan <- pResp{domain: domain, ipAddress: ipAddress, rtt: stats.MaxRtt}
	}
	pinger.Count = 1
	pinger.Timeout = time.Duration(2) * time.Second
	err = pinger.Run()
	if err != nil {
		return err
	}
	return nil
}

// lookup a domain and return the ip
func lookup(domain string) (string, error) {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return "", err
	} else if len(addrs) < 1 {
		return "", fmt.Errorf("%s doesn't resolve!!! is DNS broken or is it a bad hostname?", domain)
	} else if len(addrs) == 1 {
		return addrs[0], nil
	} else {
		// Loop through all addrs and find one that will ping (there might be stale ones)
		found := make(chan int, len(addrs)+1)
		for index, addr := range addrs {
			index, addr := index, addr
			pinger, err := ping.NewPinger(addr)
			if err != nil {
				return "", err
			}
			pinger.SetPrivileged(true)
			pinger.OnRecv = func(pkt *ping.Packet) {}
			pinger.OnFinish = func(stats *ping.Statistics) {
				if stats.MaxRtt > 0 {
					// Something is there
					found <- index
				}
			}
			pinger.Count = 1
			pinger.Timeout = time.Duration(2) * time.Second
			pinger.Run()
		}
		// If none of them ping, set the IP as the first one
		found <- 0
		return addrs[<-found], nil
	}
}

func connectToDB(dbType string, dsn string) (*sql.DB, error) {
	// connect to the database
	switch dbType {
	case "mysql", "postgres":
		db, err := sql.Open(dbType, dsn)
		if err == nil {
			err = db.Ping()
		}
		return db, err
	default:
		return nil, fmt.Errorf("unsupported database: %s", dbType)
	}
}

func database(ctx context.Context, dbType string, dsn string, data chan pResp) error {
	db, err := connectToDB(dbType, dsn)
	if err != nil {
		return err
	}
	// prepare the query outside the loop
	var prepareStr string
	switch dbType {
	case "mysql":
		prepareStr = "insert pings set domain = ?, packet_rtt = ?, ip_address = ?"
	case "postgres":
		prepareStr = "insert into pings (domain, packet_rtt, ip_address) values ($1,$2,$3)"
	default:
		return fmt.Errorf("unsupported database: %s", dbType)
	}

	stmt, err := db.Prepare(prepareStr)
	if err != nil {
		return err
	}

	for {
		// wait for a result from a pinger
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-data:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var err error
			var res sql.Result
			if r.rtt <= 0 {
				res, err = stmt.ExecContext(ctx, r.domain, sql.NullString{}, r.ipAddress)
			} else {
				res, err = stmt.ExecContext(ctx, r.domain, r.rtt.Seconds(), r.ipAddress)
			}
			cancel()
			if err != nil {
				log.Println(err)
				continue
			}
			_, err = res.RowsAffected()
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//get flags
	dsn := flag.String("dsn", "", "The connection string for your database")
	dbType := flag.String("dbtype", "mysql", "database to connect to")
	filename := flag.String("f", "config.yml", "YAML configuration file")
	dnsRefreshMinutes := flag.Int("dnsRefresh", 15, "minutes between dns refreshes")
	listen := flag.String("listen", ":9103", "http metrics/debug server listen address")

	flag.Parse()

	// open the config file
	configFile, err := os.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}
	// process it
	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatal(err)
	}

	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//channel that all p's will send back to the main thread on
	dbChan := make(chan pResp)
	for domain, params := range config.Domains {
		domain := domain
		delay := params.Delay
		g.Add(func() error {
			var ipAddress string
			potentialIPAddress, err := lookup(domain)
			if err != nil {
				log.Println(err)
			} else {
				ipAddress = potentialIPAddress
			}
			pingTicker := time.NewTicker(time.Duration(delay) * time.Second)
			defer pingTicker.Stop()
			dnsTicker := time.NewTicker(time.Duration(*dnsRefreshMinutes) * time.Minute)
			defer dnsTicker.Stop()
			for {
				select {
				case <-pingTicker.C:
					log.Printf("running ping for %s\n", domain)
					err := p(ctx, dbChan, domain, ipAddress)
					if err != nil {
						log.Printf("error when pinging %s: %v\n", domain, err)
					}
				case <-dnsTicker.C:
					var err error
					log.Printf("running lookup for %s\n", domain)
					potentialIPAddress, err := lookup(domain)
					if err != nil {
						log.Println(err)
					} else {
						ipAddress = potentialIPAddress
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}, func(err error) {
			log.Println(err)
		})
	}

	//loop through every response and process the input for the DB
	g.Add(func() error {
		return database(ctx, *dbType, *dsn, dbChan)
	},
		func(err error) {
			log.Println(err)
		})

	// run an http server for debugging memory leaks / performance issues and scrape internal go prometheus metrics

	g.Add(func() error {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Beginning to serve on port %s\n", *listen)
		return http.ListenAndServe(*listen, nil)
	}, func(err error) {
		log.Println(err)
	})

	if err := g.Run(); err != nil {
		log.Println("run group ended. exiting...")
	}
}
