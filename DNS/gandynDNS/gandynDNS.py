#!/usr/bin/python
""" GandynDNS - this script get your public IP and change your Gandi DNS entry 
		
		************ DEPRECATED ************

		This use the old V3 API which is now deprecated. 
		Use Gandi v5 LiveDNS instead.

		************ DEPRECATED ************

		
    Prune@lecentre.net - 2013

"""

import urllib
import xmlrpclib
import datetime
import sys

####################################################################################
# 
# config
#
####################################################################################
apiDEVurl = 'https://rpc.ote.gandi.net/xmlrpc/'
apiPRODurl = 'https://rpc.gandi.net/xmlrpc/'

apikey = ''

# publicIPprovider = 'http://api.externalip.net/ip/'
# next time, try http://ipinfo.io/
publicIPprovider = 'http://icanhazip.com'

# set the domain to use (base domain as defined in Gandi
domain='mydomain.net'

# set the name of your host. Will be myDNSname.domain
# in this example, home.mydomain.net
myDNSname = "home"

####################################################################################
# 
# do the show
#
####################################################################################
api=xmlrpclib.ServerProxy(apiPRODurl)

try :
	myIP = urllib.urlopen(publicIPprovider).read().strip()
	print "server IP is %s" % myIP
except :
	print "can't find your public IP address - check your provider in the config"
	sys.exit(1)

try :
	# get the zone ID
	domainData = api.domain.info(apikey, domain)
	zone_id = domainData['zone_id']
	zone_info = api.domain.zone.info(apikey, zone_id)

except :
	print "Error getting Gandi infos. Check your API key and network"
	sys.exit(1)

# check if the IP has changed else exit OK
current_entry = api.domain.zone.record.list(apikey, zone_id, zone_info['version'],{"name": myDNSname})
if current_entry and current_entry[0]['value'] :
	current_IP = current_entry[0]['value'].strip()
	print "current IP configured for %s is %s" % (myDNSname, current_IP)
	if current_IP == myIP :
		print "IP is already set : " + current_IP + " ; nothing to do"
		sys.exit(0)
	else :
		print "no current IP entry found or ip missmatch, create one (configured: '%s' - current: '%s' )" % (current_IP, myIP)

try :
	# copy the actual zone to a new version
	version_id = api.domain.zone.version.new(apikey, zone_id)

	# change the name of the version
	now = datetime.datetime.now()
	zoneName = "gandyndns-" + domain + "-" + now.strftime("%Y%m%d-%H%M%s")
	api.domain.zone.update(apikey, zone_id, { "name" : zoneName })
except :
	print "Error creating a new version of the zone file"
	sys.exit(1)

print "working on zone id " + str(zone_id) + "at version " + str(version_id)

try :
	# delete old entry
	api.domain.zone.record.delete(apikey, zone_id, version_id,{ "type" : [ "A", "CNAME"], "name" : myDNSname })
except :
	print "can't delete old entry - adding a new one"

try :
	# add a new entry
	newRecord = {'name': myDNSname, 'type': 'A', 'value': myIP}
	adding = api.domain.zone.record.add(apikey, zone_id, version_id, newRecord)
except :
	print "can't add the new entry - aborting"
	sys.exit(1)

try :
	# set the new zone file as default
	api.domain.zone.version.set(apikey, zone_id, version_id)
	api.domain.zone.set(apikey, domain, zone_id)
	print "zone version set to " + str(version_id) + " for domain " + str(domain)
	print "active zone version is now " +  str(api.domain.zone.info(apikey, zone_id)['version'])
except :
	print "can't set the new zone file as default. the DNS can't be changed"
	sys.exit(1)

try :
	# delete version older than 5
	api.domain.zone.version.delete(apikey, zone_id, version_id-5)
except :
	print "no old zone file to delete"

#everything is fine
sys.exit(0)
