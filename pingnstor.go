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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/oklog/run"
	"gopkg.in/yaml.v2"
)

type pResp struct {
	domain  string
	rtt     time.Duration
	nextHop bool
}

func p(ctx context.Context, dbChan chan pResp, domain string, nexthop bool) error {
	// initialize a pinger
	pinger, err := ping.NewPinger(domain)
	if err != nil {
		return err
	}
	pinger.SetPrivileged(true)
	pinger.OnRecv = func(pkt *ping.Packet) {}
	pinger.OnFinish = func(stats *ping.Statistics) {
		dbChan <- pResp{domain: domain, rtt: stats.MaxRtt, nextHop: nexthop}
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
		prepareStr = "insert pings set domain = ?, packet_rtt = ?, next_hop = ?"
	case "postgres":
		prepareStr = "insert into pings (domain, packet_rtt, next_hop) values ($1,$2,$3)"
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
				res, err = stmt.ExecContext(ctx, r.domain, sql.NullString{}, r.nextHop)
			} else {
				res, err = stmt.ExecContext(ctx, r.domain, r.rtt.Seconds(), r.nextHop)
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

	flag.Parse()

	//open the config file
	config, err := os.ReadFile(*filename)
	if err != nil {
		log.Fatal(err)
	}
	//make a map
	configMap := make(map[string]interface{})
	//parse the config
	err = yaml.Unmarshal(config, &configMap)
	if err != nil {
		log.Fatal(err)
	}

	//spawn run group
	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//channel that all p's will send back to the main thread on
	dbChan := make(chan pResp)
	for _, domains := range configMap["domains"].([]interface{}) {
		for domain, params := range domains.(map[interface{}]interface{}) {
			//type assertions
			delay := params.(map[interface{}]interface{})["delay"].(int)
			domain := domain.(string)
			g.Add(func() error {
				for {
					pingTicker := time.NewTicker(time.Duration(delay) * time.Second)
					dnsTicker := time.NewTicker(time.Duration(*dnsRefreshMinutes) * time.Minute)
					select {
					case <-pingTicker.C:
						log.Println("running ping for", domain)
						err := p(ctx, dbChan, domain, false)
						if err != nil {
							log.Println(err)
						}
					case <-dnsTicker.C:
						var err error
						log.Println("running lookup for", domain)
						domain, err = lookup(domain)
						if err != nil {
							log.Println(err)
						}
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}, func(err error) {
				log.Println(err)
			})
		}
	}

	//loop through every response and process the input for the DB

	g.Add(func() error {
		return database(ctx, *dbType, *dsn, dbChan)
	},
		func(err error) {
			log.Println(err)
		})

	if err := g.Run(); err != nil {
		log.Println("run group ended. exiting...")
	}
}
