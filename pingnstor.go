package main

import (
	"database/sql"
	"flag"
	"log"
	"net"
	"time"

	"github.com/jrcichra/pingnstor/traceroute"

	"io/ioutil"

	"github.com/go-ping/ping"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"
)

type pResp struct {
	domain  string
	rtt     time.Duration
	nextHop bool
}

func p(dbChan chan pResp, sleepChan chan bool, refreshChan chan string, site string, nexthop bool) {
	domain := site
	done := false
	for !done {
		done = !<-sleepChan //let the sleeper decide if we should wait or not on start
		//see if we should update our site, if the channel is empty, do nothing
		select {
		case d := <-refreshChan:
			site = d
		default:
		}
		// initalize a pinger
		pinger, err := ping.NewPinger(site)
		if err != nil {
			log.Printf("WARN: %s\n", err.Error())
			continue
		}
		pinger.SetPrivileged(true)

		pinger.OnRecv = func(pkt *ping.Packet) {
			// log.Println(site, "got an onRecv")
		}
		pinger.OnFinish = func(stats *ping.Statistics) {
			// log.Println(site, "got an onFinish")
			// log.Println("stats for site", site, ":", stats)
			dbChan <- pResp{domain: domain, rtt: stats.MaxRtt, nextHop: nexthop}
		}
		pinger.Count = 1
		pinger.Timeout = time.Duration(2) * time.Second

		//ping until our sleeper tells us otherwise
		// log.Println("I am pinging", site)
		pinger.Run()
		// log.Println(site, "is done pinging")
	}
}

//lookup a domain and return the ip
func lookup(domain string) string {
	ret := ""
	addrs, err := net.LookupHost(domain)
	if err != nil {
		log.Println(err)
	} else if len(addrs) < 1 {
		log.Println(domain, "doesn't resolve!!! is DNS broken or is it a bad hostname?")
	} else if len(addrs) == 1 {
		ret = addrs[0]
	} else {
		//Loop through all addrs and find one that will ping (aka there might be stale ones)
		//If none of them ping, set the IP as the first one found
		next := make(chan bool)
		foundIndex := 0
		for index, addr := range addrs {
			pinger, err := ping.NewPinger(addr)
			if err != nil {
				log.Printf("WARN: %s\n", err.Error())
				continue
			}
			pinger.SetPrivileged(true)

			pinger.OnRecv = func(pkt *ping.Packet) {
			}
			pinger.OnFinish = func(stats *ping.Statistics) {
				cont := true
				if stats.MaxRtt <= 0 {
					//No ping response, do nothing and continue on
				} else {
					//Something is there
					foundIndex = index
					cont = false
				}
				next <- cont
			}
			pinger.Count = 1
			pinger.Timeout = time.Duration(2) * time.Second

			pinger.Run()
			//break the loop if we found something before the end
			if !<-next {
				break
			}
		}
		ret = addrs[foundIndex]

	}
	return ret
}

func sleeper(sleepChan chan bool, delay int) {
	for {
		//ping upon startup, move after sleep if you want a delay first
		// log.Println(site, "'s sleeper is sending a true")
		sleepChan <- true
		// log.Println(site, "'s sleeper is done sending a true")
		// log.Println(site, "'s sleeper is sleeping for", delay, "seconds...")
		time.Sleep(time.Duration(delay) * time.Second)
		// log.Println(site, "'s sleeper is done sleeping")
	}
}

func connectToDB(dsn *string) (*sql.DB, error) {
	// connect to the database
	db, err := sql.Open("mysql", *dsn)
	if err == nil {
		err = db.Ping()
	}
	return db, err
}

func reconnectToDB(dsn *string) *sql.DB {
	var err error
	var db *sql.DB
	err = nil
	for err == nil {
		db, err = connectToDB(dsn)
		if err != nil {
			log.Println(err)
			db.Close()
			time.Sleep(time.Duration(1) * time.Second)
			err = nil
		}

	}
	return db
}

//refresh a domain on a ticker
func domainRefresh(domain string, refreshChan chan string) {
	for {
		t := time.NewTicker(10 * time.Minute)
		select {
		case <-t.C:
			//Kick off a domain refresh
			refreshChan <- lookup(domain)
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//get flags
	dsn := flag.String("dsn", "", "The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name")
	filename := flag.String("f", "config.yml", "YAML configuration file")
	hopNum := flag.Int("hopnum", 0, "Number of the hop you want to ping")
	hopInt := flag.Int("hopint", 0, "Automatically determine the next hop on startup and use this interval")

	flag.Parse()

	db, err := connectToDB(dsn)
	if err != nil {
		log.Fatal(err)
	}

	//open the config file
	config, err := ioutil.ReadFile(*filename)
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

	//spawn things

	//channel that all p's will send back to the main thread on
	dbChan := make(chan pResp)
	for _, domains := range configMap["domains"].([]interface{}) {
		for domain, params := range domains.(map[interface{}]interface{}) {

			//type assertions
			delay := params.(map[interface{}]interface{})["delay"].(int)
			domain := domain.(string)

			//spawn a sleeper and a channel which will trigger a pinger to ping, which in turn triggers the DB
			//giving each pinger its own sleeper allows for per-domain sleeps, and because in go this is easy
			sleepChan := make(chan bool) //true=keep pinging, false=last ping and die
			go sleeper(sleepChan, delay)
			//spawn a domain refresher that will occassionally do a DNS lookup for long running pingnstors
			refreshChan := make(chan string)
			go domainRefresh(domain, refreshChan)
			//spawn a pinger with a delay for this
			go p(dbChan, sleepChan, refreshChan, domain, false)

		}
	}

	//if they want their next hop recorded as well, do that here
	if *hopNum > 0 && *hopInt > 0 {
		to := traceroute.TracerouteOptions{}
		to.SetRetries(3) //Third times a charm
		to.SetMaxHops(*hopNum + 1)
		tr, err := traceroute.Traceroute("8.8.8.8", &to)
		if err != nil {
			log.Println(err)
		}
		log.Println("The hops I see: ", tr.Hops)
		//spawn a sleeper and a channel which will trigger a pinger to ping, which in turn triggers the DB
		//giving each pinger its own sleeper allows for per-domain sleeps, and because in go this is easy
		sleepChan := make(chan bool) //true=keep pinging, false=last ping and die
		go sleeper(sleepChan, *hopInt)
		//spawn a domain refresher that will occassionally do a DNS lookup for long running pingnstors
		refreshChan := make(chan string)
		domain := tr.Hops[*hopNum-1].AddressString()
		go domainRefresh(domain, refreshChan)
		//spawn a pinger with a delay for this
		go p(dbChan, sleepChan, refreshChan, domain, true)
	}

	//loop through every response and process the input for the DB

	// prepare the query outside the loop
	stmt, err := db.Prepare("insert pings set domain = ?, packet_rtt = ?, next_hop = ?")
	if err != nil {
		log.Println(err)
		//reconnect to the db
		db.Close()
		db = reconnectToDB(dsn)
	}

	for {
		//wait for a result from a pinger
		r := <-dbChan
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
			//reconnect to the db
			db.Close()
			db = reconnectToDB(dsn)
			continue
		}
		_, err = res.RowsAffected()
		if err != nil {
			log.Println(err)
			//reconnect to the db
			db.Close()
			db = reconnectToDB(dsn)
			continue
		}

	}
}
