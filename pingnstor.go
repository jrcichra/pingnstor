package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"io/ioutil"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sparrc/go-ping"
	"gopkg.in/yaml.v2"
)

type pResp struct {
	domain string
	rtt    time.Duration
}

func p(dbChan chan pResp, sleepChan chan bool, site string) {

	// initalize a pinger
	pinger, err := ping.NewPinger(site)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	pinger.SetPrivileged(true)

	pinger.OnRecv = func(pkt *ping.Packet) {

	}
	pinger.OnFinish = func(stats *ping.Statistics) {
		dbChan <- pResp{domain: site, rtt: stats.MaxRtt}
	}
	pinger.Count = 1

	//ping until our sleeper tells us otherwise

	done := false
	for !done {
		done = !<-sleepChan //let the sleeper decide if we should wait or not on start, the not is for keeping the done logic here sane,
		//as a true coming over makes more sense than a false to keep going
		pinger.Run()
		fmt.Println("I am pinging", site)

	}
}

func sleeper(sleepChan chan bool, delay int) {
	for {
		//ping upon startup, move after sleep if you want a delay first
		sleepChan <- true
		fmt.Println("Sleeping for", delay, "seconds...")
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func main() {
	//get flags
	dsn := flag.String("dsn", "", "The connect string for your database - see https://github.com/go-sql-driver/mysql#dsn-data-source-name")
	filename := flag.String("f", "config.yml", "YAML configuration file")

	flag.Parse()

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

	// connect to the database
	db, err := sql.Open("mysql", *dsn)
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

			//debug prints
			//fmt.Printf("Domain:%s\n", domain)
			//fmt.Printf("Delay:%d\n", delay)

			//spawn a sleeper and a channel which will trigger a pinger to ping, which in turn triggers the DB
			//giving each pinger its own sleeper allows for per-domain sleeps, and because in go this is easy
			sleepChan := make(chan bool) //true=keep pinging, false=last ping and die
			go sleeper(sleepChan, delay)
			//spawn a pinger with a delay for this
			go p(dbChan, sleepChan, domain)

		}

	}
	//loop through every response and process the input for the DB
	for {
		//wait for a result from a pinger
		r := <-dbChan
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
