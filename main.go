package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/robfig/cron"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	ipDomains = []string{"http://ipinfo.io/ip", "http://myexternalip.com/raw"}

	ctx    = context.TODO()
	client *godo.Client
)

func main() {

	var key = os.Getenv("DO_DDNS_KEY")
	if key == "" {
		fmt.Println("Missing Digital Ocean key")
		os.Exit(1)
	}

	// Get local records
	b, err := ioutil.ReadFile("records.yaml")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var records []record
	err = yaml.Unmarshal(b, &records)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Println("No records to update")
		os.Exit(1)
	}

	for _, record := range records {
		if record.Domain == "" {
			fmt.Println("Domain is required")
			os.Exit(1)
		}
	}

	// Get live records
	oauthClient := oauth2.NewClient(context.Background(), &TokenSource{AccessToken: key})
	client = godo.NewClient(oauthClient)

	// client.Domains.EditRecord()

	// Update every 30 mins
	c := cron.New()
	err = c.AddFunc("0 30 * * * *", updateIP)
	if err != nil {
		fmt.Println(err)
	}
	c.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func updateIP() {

}

type record struct {
	Domain   string `yaml:"domain"`
	Type     string `yaml:"type"`
	Hostname string `yaml:"hostname"`
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
