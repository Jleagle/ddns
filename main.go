package main

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/robfig/cron"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	ipDomains    = []string{"http://ipinfo.io/ip", "http://myexternalip.com/raw", "https://ifconfig.co/ip"}
	localRecords []record

	// Digital Ocean
	ctx    = context.TODO()
	client *godo.Client
)

func init() {

	// Get Digital Ocean key
	var key = os.Getenv("KEY")
	if key == "" {
		fmt.Println("Missing Digital Ocean key")
		os.Exit(1)
	}

	// Make Didital Ocean client
	oauthClient := oauth2.NewClient(context.Background(), &TokenSource{AccessToken: key})
	client = godo.NewClient(oauthClient)

	// Get local records
	b, err := ioutil.ReadFile("records.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(b, &localRecords)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(localRecords) == 0 {
		fmt.Println("No records to update")
		os.Exit(1)
	}

	for _, record := range localRecords {
		if record.Domain == "" {
			fmt.Println("Domain in YAML is required")
			os.Exit(1)
		}
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
		err := c.AddFunc("0 30 * * * *", updateIP)
		if err != nil {
			fmt.Println(err)
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
	for _, v := range ipDomains {

		resp, err := http.Get(v)
		if err != nil {
			fmt.Println(err)
			continue
		}
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			continue
		}

		ip = strings.TrimSpace(string(bytes))
		break
	}

	if ip == "" {
		fmt.Println("Could not fetch IP")
		return
	}

	fmt.Println("IP is " + ip)

	// Get domains
	domains, _, err := client.Domains.List(ctx, &godo.ListOptions{Page: 1, PerPage: 1000})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, localRecord := range localRecords {
		for _, domain := range domains {
			if domain.Name == localRecord.Domain {

				// Get domain records
				records, _, err := client.Domains.Records(ctx, domain.Name, &godo.ListOptions{Page: 1, PerPage: 1000})
				if err != nil {
					fmt.Println("Failed to get records for " + domain.Name + ": " + err.Error())
				}

				for _, liveRecord := range records {
					if liveRecord.Name == localRecord.SubDomain && liveRecord.Type == "A" {

						_, _, err := client.Domains.EditRecord(ctx, domain.Name, liveRecord.ID, &godo.DomainRecordEditRequest{Data: ip})
						if err != nil {
							fmt.Println("Failed to update record: " + err.Error())
						}
					}
				}
			}
		}
	}
}

type record struct {
	Domain    string `yaml:"domain"`
	SubDomain string `yaml:"subdomain"`
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
