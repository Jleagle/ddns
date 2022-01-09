package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Jleagle/ddns/providers"
	"github.com/jpillora/go-tld"
	"github.com/robfig/cron"
	"gopkg.in/yaml.v2"
)

const (
	providerCloudflare   = "cloudflare"
	providerDigitalOcean = "digitalocean"
)

var (
	flagRecordsFile  = flag.String("file", "records.yaml", "The file with records to update")
	flagDryRunMode   = flag.Bool("dry", false, "Dry run mode")
	flagOnLoad       = flag.Bool("onload", false, "Run when the program starts")
	flagCron         = flag.Bool("cron", false, "Run forever on a cron")
	flagCronDuration = flag.String("duration", "30m", "Cron intervals")

	logger       = log.New(os.Stdout, "DDNS: ", log.LstdFlags)
	recordsCache = map[providerEnum][]string{}
)

type provider interface {
	GetDomainID(domain string) (domainID string, err error)
	GetRecordID(domainID, name string) (recordID interface{}, err error)
	EditRecord(domainID string, recordID interface{}, ip string) (err error)
}

type providerEnum string

func (p providerEnum) getProvider() provider {
	switch p {
	case providerDigitalOcean:
		return providers.DigitalOcean{}
	case providerCloudflare:
		return providers.Cloudflare{}
	default:
		return nil
	}
}

func main() {

	flag.Parse()

	// Read domains from config
	b, err := ioutil.ReadFile(*flagRecordsFile)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(b, &recordsCache)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if len(recordsCache) == 0 {
		logger.Println("No records to update")
		os.Exit(1)
	}

	if !*flagOnLoad && !*flagCron {
		logger.Println("DDNS needs to run on load or on cron")
		os.Exit(1)
	}

	//
	if *flagOnLoad {
		updateIP()
	}

	if *flagCron {

		c := cron.New()
		err := c.AddFunc("@every "+*flagCronDuration, updateIP)
		if err != nil {
			logger.Println(err)
		}
		c.Start()

		var wg sync.WaitGroup
		wg.Add(1)
		wg.Wait()
	}
}

var (
	cachedIP    = ""
	ipProviders = []string{
		"https://ipinfo.io/ip",
		"https://myexternalip.com/raw",
		"https://ifconfig.co/ip",
	}
)

func updateIP() {

	// Get current IP
	var ip string
	var err error

	for _, v := range ipProviders {

		var resp *http.Response
		var bytes []byte

		resp, err = http.Get(v)
		if err != nil {
			continue
		}

		bytes, err = ioutil.ReadAll(resp.Body)
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

	// Cache to stop unnecessary api calls
	if ip == cachedIP {
		return
	}

	cachedIP = ip

	logger.Println("IP is " + ip)

	// Get domains
	for providerEnum, domains := range recordsCache {

		provider := providerEnum.getProvider()

		for _, domain := range domains {

			parsed, err := tld.Parse("https://" + domain)
			if err != nil {
				logger.Println(err)
				continue
			}

			domainID, err := provider.GetDomainID(parsed.Domain + "." + parsed.TLD)
			if err != nil {
				logger.Println(err)
				continue
			}

			recordID, err := provider.GetRecordID(domainID, domain)
			if err != nil {
				logger.Println(err)
				continue
			}

			logger.Printf("Updating %s on %s", domain, providerEnum)

			if !*flagDryRunMode {
				err := provider.EditRecord(domainID, recordID, ip)
				if err != nil {
					logger.Println("Failed to update record: " + err.Error())
				}
			}
		}
	}
}
