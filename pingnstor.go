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

func connectToDB(dsn string) (*sql.DB, error) {
	// connect to the database
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		err = db.Ping()
	}
	return db, err
}

func database(ctx context.Context, dsn string, data chan pResp) error {
	db, err := connectToDB(dsn)
	if err != nil {
		log.Fatal(err)
	}
	// prepare the query outside the loop
	stmt, err := db.Prepare("insert pings set domain = ?, packet_rtt = ?, next_hop = ?")
	if err != nil {
		log.Println(err)
	}

	for {
		//wait for a result from a pinger
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-data:
			var err error
			var res sql.Result
			ns := sql.NullString{}
			if r.rtt <= 0 {
				res, err = stmt.Exec(r.domain, ns, r.nextHop)
			} else {
				res, err = stmt.Exec(r.domain, r.rtt.Seconds(), r.nextHop)
			}
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
	dsn := flag.String("dsn", "", "The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name")
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
		return database(ctx, *dsn, dbChan)
	},
		func(err error) {
			log.Println(err)
		})

	if err := g.Run(); err != nil {
		log.Println("run group ended. exiting...")
	}
}
