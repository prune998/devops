# gandynDNS

## gandi-live-dns.py
This tool use the Gandi V5 API to update your DNS automaticaly. This is a dyndns using Gandi domain name provider

### Usage

Edit the script to your needs, add your key and domain names in the `config` section:

```python
####################################################################################
#
# config
#
####################################################################################
api_endpoint = 'https://dns.api.gandi.net/api/v5'

api_secret = 'CHANGE ME'

# publicIPprovider = 'http://api.externalip.net/ip/'
# next time, try http://ipinfo.io/
publicIPprovider = 'http://icanhazip.com'

# set the domain to use (base domain as defined in Gandi)
domain='example.net'

# set the name of your host. Will be subdomains[0].domain
# Why ? don't know...
subdomains = ["myserver"]
ttl = '300'
```

Where:
- `api_secret`: the API secret key you get from the Gandi admin interface
- `domain`: the base domain name you want to use, which must exist in Gandi
- `subdomains`: add only one name, which will be your host

## gandynDNS.py
This tool use the Gandi V3/V4 API to update your DNS automaticaly. This is a dyndns using Gandi domain name provider

**THIS TOOL IS DEPRECATED**
