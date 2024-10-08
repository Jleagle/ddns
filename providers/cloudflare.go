package providers

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/cloudflare/cloudflare-go"
	"golang.org/x/net/context"
)

var cloudflareClient *cloudflare.API
var cloudflareLock sync.Mutex

func getCloudflareClient() (*cloudflare.API, error) {

	cloudflareLock.Lock()
	cloudflareLock.Unlock()

	if cloudflareClient == nil {

		var key = os.Getenv("CF_KEY")
		if key == "" {
			return nil, errors.New("missing cloudflare key")
		}

		api, err := cloudflare.NewWithAPIToken(key)
		if err != nil {
			return nil, err
		}

		cloudflareClient = api
	}

	return cloudflareClient, nil
}

type Cloudflare struct {
}

func (c Cloudflare) GetDomainID(domain string) (string, error) {

	api, err := getCloudflareClient()
	if err != nil {
		return "", err
	}

	return api.ZoneIDByName(domain)
}

func (c Cloudflare) GetRecordID(domainID, name string) (interface{}, error) {

	api, err := getCloudflareClient()
	if err != nil {
		return "", err
	}

	records, _, err := api.ListDNSRecords(
		context.Background(),
		cloudflare.ZoneIdentifier(domainID),
		cloudflare.ListDNSRecordsParams{Type: "A", Name: name},
	)

	if len(records) == 1 {
		return records[0].ID, nil
	}

	return "", fmt.Errorf("no records matching %s", name)
}

func (c Cloudflare) EditRecord(domainID string, recordID interface{}, ip string) error {

	api, err := getCloudflareClient()
	if err != nil {
		return err
	}

	_, err = api.UpdateDNSRecord(
		context.Background(),
		cloudflare.ZoneIdentifier(domainID),
		cloudflare.UpdateDNSRecordParams{Content: ip, ID: recordID.(string)},
	)

	return err
}
