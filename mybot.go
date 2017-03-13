

package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: mybot slack-bot-token\n")
		os.Exit(1)
	}

	// start a websocket-based Real Time API session
	ws, id := slackConnect(os.Args[1])
	fmt.Println("mybot ready, ^C exits")

	for {
		// read each incoming message
		m, err := getMessage(ws)
		if err != nil {
			log.Fatal(err)
		}

		// see if we're mentioned
		if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+id+">") {
			// if so try to parse if
			parts := strings.Fields(m.Text)
			if len(parts) == 3 && parts[1] == "quote" {
				// looks good, get the quote and reply with the result
				go func(m Message) {
					m.Text = getQuote(parts[2])
					postMessage(ws, m)
				}(m)
				// NOTE: the Message object is copied, this is intentional
			} else if parts[1] == "hi" {
				m.Text = fmt.Sprintf("Welcome to gostock! I love pizza! Look up a stock price by using the commands: '@gostock quote (your stock symbol)'")
				postMessage(ws, m)
			} else if parts[1] == "buy" && len(parts) == 3 {

				m.Text = fmt.Sprintf("You want to buy: " + parts[2] + " " + getQuote(parts[2]))
				var sStmt string = "insert into portfolios (stock_id, stock_price, date) values ($1, $2, $3)"
				    url := "postgres://dnnzypcfamjdvv:81109122c1678fcb5290bc0e3267de8aa77ba58813dabe6697e7455fc4a9f30e@ec2-54-225-104-61.compute-1.amazonaws.com:5432/ddei9gtbsqf3tl"
					connection, _ := pq.ParseURL(url)
					connection += " sslmode=require"
				db, err := sql.Open("postgres", connection)
				if err != nil {
					log.Fatal(err)
				}

				stmt, err := db.Prepare(sStmt)
				if err != nil {
					log.Fatal(err)
				}

				res, err := stmt.Exec(parts[2], getPrice(parts[2]), time.Now())
				if err != nil || res == nil {
					log.Fatal(err)
				}
				stmt.Close()
				db.Close()

				postMessage(ws, m)
			} else {
				// huh?
				m.Text = fmt.Sprintf("sorry, that does not compute\n")
				postMessage(ws, m)
			}
		}
	}
}

func getQuote(sym string) string {
	sym = strings.ToUpper(sym)
	url := fmt.Sprintf("http://download.finance.yahoo.com/d/quotes.csv?s=%s&f=nsl1op&e=.csv", sym)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if len(rows) >= 1 && len(rows[0]) == 5 {
		return fmt.Sprintf("%s (%s) is trading at $%s", rows[0][0], rows[0][1], rows[0][2])
	}
	return fmt.Sprintf("unknown response format (symbol was \"%s\")", sym)
}

func getPrice(sym string) string {
	sym = strings.ToUpper(sym)
	url := fmt.Sprintf("http://download.finance.yahoo.com/d/quotes.csv?s=%s&f=nsl1op&e=.csv", sym)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if len(rows) >= 1 && len(rows[0]) == 5 {
		return fmt.Sprintf("%s", rows[0][2])
	}
	return fmt.Sprintf("unknown response format (symbol was \"%s\")", sym)
}
