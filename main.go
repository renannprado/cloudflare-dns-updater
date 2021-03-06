package main

import (
	"flag"
	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	iface_prefix := flag.String("iface-prefix", "", "name or prefix of the network interface you want to watch")
	cloudflare_email := flag.String("cloudflare-email", "", "name or prefix of the network interface you want to watch")
	cloudflare_api_token := flag.String("cloudflare-api-token", "", "name or prefix of the network interface you want to watch")
	cloudflare_dns_zone := flag.String("cloudflare-dns-zone", "", "name or prefix of the network interface you want to watch")
	cloudflare_dns_record_name := flag.String("cloudflare-dns-record-name", "", "name or prefix of the network interface you want to watch")
	wait_between_cycles := flag.Duration("check-interval", time.Second*30, "how often the IP is checked for change")

	flag.Parse()

	if !flag.Parsed() {
		flag.Usage()
		return
	}

	cloudflareApi, err := cloudflare.New(*cloudflare_api_token, *cloudflare_email)
	if err != nil {
		log.Fatalf("failed to createa clouflare client: %+v\n", err)
	}

	for {
		ipv6, err := findIPv6(*iface_prefix)

		if err != nil {
			log.Printf("failed to find ipv6 for interface %s: %+v\n", *iface_prefix, err)
		} else {
			err = upsertCloudflareDNS(cloudflareApi, *cloudflare_dns_zone, *cloudflare_dns_record_name, ipv6)

			if err != nil {
				log.Printf("error while trying to update cloudflare DNS %s to %s: %+v\n", *cloudflare_dns_record_name, ipv6, err)
			}
		}

		log.Printf("sleeping for %s\n", *wait_between_cycles)

		time.Sleep(*wait_between_cycles)
	}
}

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

func upsertCloudflareDNS(cloudflareApi *cloudflare.API, dnsZone string, name string, ip string) error {
	dnsZoneId, err := cloudflareApi.ZoneIDByName(dnsZone)

	if err != nil {
		return errors.Wrapf(err, "failed to get DNS zone ID for %s", dnsZoneId)
	}

	existingDns, err := cloudflareApi.DNSRecords(dnsZoneId, cloudflare.DNSRecord{
		Type: "AAAA",
		Name: name,
	})

	if err != nil {
		return errors.Wrapf(err, "failed to get existing DNS record %s", dnsZoneId)
	}

	newDnsRecord := cloudflare.DNSRecord{
		Type:    "AAAA",
		Name:    name,
		Content: ip,
		TTL:     300,
	}

	switch len(existingDns) {
	case 0:
		{
			_, err := cloudflareApi.CreateDNSRecord(dnsZoneId, newDnsRecord)

			log.Printf("attempting to create DNS record %s with content %s\n", name, ip)

			if err != nil {
				return errors.Wrap(err, "failed to create new DNS record")
			}

			log.Printf("DNS record created with success")

			return nil
		}
	case 1:
		{
			if existingDns[0].Content == ip {
				log.Println("IP is already up to date in cloudflare, nothing to do")
				return nil
			}

			log.Printf("old IP was %s, updating to %s\n", existingDns[0].Content, newDnsRecord.Content)

			err = cloudflareApi.UpdateDNSRecord(dnsZoneId, existingDns[0].ID, newDnsRecord)

			if err != nil {
				return errors.Wrapf(err, "failed to updated DNS record %s to %s", name, ip)

			}

			log.Printf("DNS record updated with success")

			return nil
		}
	default:
		return errors.Errorf("expected 0 or 1 DNS record with name %s but found %d", name, len(existingDns))
	}
}
