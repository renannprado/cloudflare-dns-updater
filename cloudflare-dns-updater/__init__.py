import netifaces
import re
import os


# iface_prefix = os.environ['IFACE_PREFIX']
# cloudflare_email = os.environ['CLOUDFLARE_EMAIL']
# cloudflare_api_token = os.environ['CLOUDFLARE_API_TOKEN']
# cloudflare_dns_zone = os.environ['CLOUDFLARE_DNS_ZONE_ID']
# cloudflare_ = os.environ['CLOUDFLARE_DNS_IDENTIFIER']
# cloudflare_dns = os.environ['CLOUDFLARE_DNS']
# last_ip_file = os.environ['LAST_IP_FILE']
# wait_between_cycles = os.environ['MAX_WAIT_TIME']
def find_interface_prefix(prefix):
  filter_regex = re.compile(f"^{prefix}.*")
  return list(filter(filter_regex.match, netifaces.interfaces()))[0]

def upsert_cloudflare_dns(email, token, zone_id, dns):
  print(' faz nada')

iface_name = find_interface_prefix("wl")
print(iface_name)
addresses = netifaces.ifaddresses(iface_name)[netifaces.AF_INET6]
print(addresses)

for i in addresses:
  print(i)