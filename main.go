package main

import (
	"flag"
	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func findIPv6(ifacePrefix string) (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "failed to read network interfaces")
	}
	for _, iface := range ifaces {
		if !strings.HasPrefix(iface.Name, ifacePrefix) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", errors.Wrapf(err, "failed to read addresses from interface %s", iface.Name)
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if toIPv4 := v.IP.To4(); toIPv4 == nil {
					ipv6 := v.IP.To16()
					// cannot be a link local one
					if ipv6.IsLinkLocalMulticast() || ipv6.IsLinkLocalUnicast() {
						continue
					}

					return ipv6.String(), nil
				}
			}
		}

		return "", errors.Errorf("could not find ipv6 address for interface %s", iface.Name)
	}

	return "", errors.Errorf("could not find any network interface matching with %s", ifacePrefix)
}

func main() {
	iface_prefix := flag.String("iface-prefix", "", "name or prefix of the network interface you want to watch")
	cloudflare_email := flag.String("cloudflare-email", "", "name or prefix of the network interface you want to watch")
	cloudflare_api_token := flag.String("cloudflare-api-token", "", "name or prefix of the network interface you want to watch")
	cloudflare_dns_zone_id := flag.String("cloudflare-dns-zone-id", "", "name or prefix of the network interface you want to watch")
	cloudflare_dns_record_name := flag.String("cloudflare-dns-record-name", "", "name or prefix of the network interface you want to watch")
	last_ip_file := flag.String("last-ip-file", "", "name or prefix of the network interface you want to watch")
	wait_between_cycles := flag.Duration("check-interval", time.Second*30, "how often the IP is checked for change")

	flag.Parse()

	if !flag.Parsed() {
		flag.Usage()
		return
	}

	cloudflareApi, err := cloudflare.New(*cloudflare_api_token, *cloudflare_email)
	if err != nil {
		log.Fatal("failed to createa clouflare client: %+v", err)
	}

	for {
		ipv6, err := findIPv6(*iface_prefix)

		if err != nil {
			log.Fatalf("failed to find ipv6 for interface %s: %+v", *iface_prefix, err)
		}

		changed, err := hasIPChanged(ipv6, *last_ip_file)

		if err != nil {
			log.Fatalf("failed to check if IP changed: %+v", err)
		}

		if !changed {
			log.Println("IP has not changed, nothing to do")
		} else {
			log.Printf("attempting to change DNS %s to %s", *cloudflare_dns_record_name, ipv6)

			err = upsertCloudflareDNS(cloudflareApi, *cloudflare_dns_zone_id, *cloudflare_dns_record_name, ipv6)

			if err != nil {
				log.Printf("error while trying to update cloudflare DNS %s to %s: %+v", cloudflare_dns_record_name, ipv6, err)
			}
		}

		log.Println("sleeping for %s", *wait_between_cycles)

		time.Sleep(*wait_between_cycles)
	}
}

func hasIPChanged(currentIPv6 string, lastIPFile string) (bool, error) {
	oldIPv6, err := ioutil.ReadFile(lastIPFile)

	if err != nil {
		if !os.IsNotExist(err) {
			return false, errors.Wrapf(err, "failed to open file %s", lastIPFile)
		}
		// file doesn exist
		f, err := os.Create(lastIPFile)

		if err != nil {
			return false, errors.Wrapf(err, "failed to open file %s", lastIPFile)
		}

		defer f.Close()

		_, err = f.WriteString(currentIPv6)

		if err != nil {
			return false, errors.Wrapf(err, "failed to write to file %s", lastIPFile)
		}

		return true, errors.Wrap(err, "failed to ")
	} else if currentIPv6 == string(oldIPv6) {
		return false, nil
	} else {
		return true, nil
	}
}

func upsertCloudflareDNS(cloudflareApi *cloudflare.API, dnsZone string, name string, ip string) error {
	dnsZone, err := cloudflareApi.ZoneIDByName(dnsZone)

	if err != nil {
		return errors.Wrapf(err, "failed to get DNS zone ID for donze %s", dnsZone)
	}

	existingDns, err := cloudflareApi.DNSRecords(dnsZone, cloudflare.DNSRecord{
		Type: "AAAA",
		Name: name,
	})

	if err != nil {
		return errors.Wrapf(err, "failed to get existing DNS record %s", dnsZone)
	}

	if len(existingDns) != 1 {
		return errors.Errorf("expected 1 DNS record with name %s but found %d", name, len(existingDns))
	}

	newDnsRecord := cloudflare.DNSRecord{
		Type:    "AAAA",
		Name:    name,
		Content: ip,
		TTL:     300,
	}

	log.Printf("old IP was %s, updating to %s\n", existingDns[0].Content, newDnsRecord.Content)

	err = cloudflareApi.UpdateDNSRecord(dnsZone, existingDns[0].ID, newDnsRecord)

	return errors.Wrapf(err, "failed to updated DNS record %s to %s", name, ip)
}
