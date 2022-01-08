package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var digitalOceanClient *godo.Client
var digitalOceanLock sync.Mutex

func getDigitalOceanClient() (*godo.Client, error) {

	digitalOceanLock.Lock()
	digitalOceanLock.Unlock()

	if digitalOceanClient == nil {

		var key = os.Getenv("DO_KEY")
		if key == "" {
			return nil, errors.New("Missing Digital Ocean key")
		}

		oauthClient := oauth2.NewClient(context.Background(), &TokenSource{AccessToken: key})
		digitalOceanClient = godo.NewClient(oauthClient)
	}

	return digitalOceanClient, nil
}

type DigitalOcean struct {
}

func (d DigitalOcean) GetDomainID(domain string) (string, error) {
	return domain, nil
}

func (d DigitalOcean) GetRecordID(domainID, name string) (interface{}, error) {

	api, err := getDigitalOceanClient()
	if err != nil {
		return nil, err
	}

	page := &godo.ListOptions{Page: 1, PerPage: 1}

	records, _, err := api.Domains.RecordsByTypeAndName(context.Background(), domainID, "A", name, page)
	if len(records) == 1 {
		return records[0].ID, nil
	}

	return nil, fmt.Errorf("no records matching %s", name)
}

func (d DigitalOcean) EditRecord(domainID string, recordID interface{}, ip string) error {

	api, err := getDigitalOceanClient()
	if err != nil {
		return err
	}

	_, _, err = api.Domains.EditRecord(context.Background(), domainID, recordID.(int), &godo.DomainRecordEditRequest{Data: ip})
	return err
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: t.AccessToken}, nil
}
