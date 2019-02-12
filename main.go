package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/robfig/cron"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	ipDomains    = []string{"http://ipinfo.io/ip", "http://myexternalip.com/raw", "https://ifconfig.co/ip"}
	localRecords = map[string][]string{}
	logger       = log.New(os.Stdout, "DDNS: ", log.LstdFlags)

	// Digital Ocean
	ctx    = context.TODO()
	client *godo.Client
)

func init() {

	// Get Digital Ocean key
	var key = os.Getenv("KEY")
	if key == "" {
		logger.Println("Missing Digital Ocean key")
		os.Exit(1)
	}

	// Make Didital Ocean client
	oauthClient := oauth2.NewClient(context.Background(), &TokenSource{AccessToken: key})
	client = godo.NewClient(oauthClient)

	// Get local records
	b, err := ioutil.ReadFile("records.yaml")
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(b, &localRecords)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if len(localRecords) == 0 {
		logger.Println("No records to update")
		os.Exit(1)
	}
}

func main() {

	oneTime, _ := strconv.ParseBool(os.Getenv("ONE_TIME"))
	onLoad, _ := strconv.ParseBool(os.Getenv("ON_LOAD"))

	if oneTime || onLoad {
		updateIP()
	}

	// Update every 30 mins
	if !oneTime {

		c := cron.New()
		err := c.AddFunc("@every 30m", updateIP)
		if err != nil {
			logger.Println(err)
		}
		c.Start()

		var wg sync.WaitGroup
		wg.Add(1)
		wg.Wait()
	}
}

func updateIP() {

	// Get current IP
	var ip string
	var err error

	for _, v := range ipDomains {

		resp, err := http.Get(v)
		if err != nil {
			continue
		}
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip = strings.TrimSpace(string(bytes))
		break
	}

	if ip == "" {
		var errString string
		if err != nil {
			errString = err.Error()
		}
		logger.Println("Could not fetch IP: " + errString)
		return
	}

	logger.Println("IP is " + ip)

	// Get domains
	liveDomains, _, err := client.Domains.List(ctx, &godo.ListOptions{Page: 1, PerPage: 1000})
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	for localDomain, localSubs := range localRecords {
		for _, liveDomain := range liveDomains {
			if localDomain == liveDomain.Name {

				// Get live records
				records, _, err := client.Domains.Records(ctx, liveDomain.Name, &godo.ListOptions{Page: 1, PerPage: 1000})
				if err != nil {
					logger.Println("Failed to get records for " + liveDomain.Name + ": " + err.Error())
				}

				for _, localSub := range localSubs {
					for _, liveSub := range records {
						if localSub == liveSub.Name && liveSub.Type == "A" {

							_, _, err := client.Domains.EditRecord(ctx, liveDomain.Name, liveSub.ID, &godo.DomainRecordEditRequest{Data: ip})
							if err != nil {
								logger.Println("Failed to update record: " + err.Error())
							}
						}
					}
				}
			}
		}
	}
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}
