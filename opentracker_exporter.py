#!/usr/bin/env python3
import os
import requests
import time
from lxml import etree
from prometheus_client import start_http_server, Counter, Gauge


# currying in python looks … wrong (but needed for map)
def parse_attrlist_item(attribute):
    def parse_attrlist_item_item(item):
        key = item.xpath(f'@{attribute}')[0]
        value = int(item.text)
        return key, value
    return parse_attrlist_item_item


class OpentrackerStats:
    def __init__(self, url):
        self.url = f'http://{url}/stats?mode=everything'
        self.update()

    def update(self):
        self.data = etree.fromstring(
            requests.get(self.url).text.encode('utf-8')
        )

        # basic stats
        self.uptime    = self.__get_value('uptime')
        self.torrents  = self.__get_value('torrents/count_mutex')
        self.peers     = self.__get_value('peers/count')
        self.seeds     = self.__get_value('seeds/count')
        self.completed = self.__get_value('completed/count')

        # connection stats
        self.connections = {
            'tcp': {
                'accept':   self.__get_value('connections/tcp/accept'),
                'announce': self.__get_value('connections/tcp/announce'),
                'scrape':   self.__get_value('connections/tcp/scrape')
            },
            'udp': {
                'overall':   self.__get_value('connections/udp/overall'),
                'connect':   self.__get_value('connections/udp/connect'),
                'announce':  self.__get_value('connections/udp/announce'),
                'scrape':    self.__get_value('connections/udp/scrape'),
                'missmatch': self.__get_value('connections/udp/missmatch')
            },
            'livesync': self.__get_value('connections/livesync/count')
        }

        # “debug” stats
        self.renew = dict(
            [(int(interval), count) for interval, count in map(
                parse_attrlist_item('interval'),
                self.data.xpath('debug/renew/count')
            )]
        )

        self.http_error = dict(
            map(
                parse_attrlist_item('code'),
                self.data.xpath('debug/http_error/count')
            )
        )

        self.mutex_stall = self.__get_value('debug/mutex_stall/count')

    def __get_value(self, expression):
        return int(self.data.xpath(f'{expression}/text()')[0])


try:
    refresh_interval = os.environ['OPENTRACKER_REFRESH']
except KeyError:
    refresh_interval = 5

tracker_url = os.environ['OPENTRACKER_URL']
stats = OpentrackerStats(tracker_url)

uptime    = Counter('tracker_uptime', 'Second the tracker has been up')
torrents  = Gauge('tracker_torrents', 'Number of tracked torrents')
peers     = Gauge('tracker_peers', 'Number of known peers')
seeds     = Gauge('tracker_seeds', 'Number of known seeds')
completed = Counter('tracker_completed', 'Number of completed torrents')

connections          = Counter('tracker_connections', 'Number of connections', ['protocol', 'type'])
connections_livesync = Counter('tracker_connections_livesync', 'Number of livesync connections')

renew       = Gauge('tracker_renew', 'Number of clients renewing the connection at a specific interval', ['interval'])
http_error  = Counter('tracker_http_error', 'Number of http errors', ['code'])
mutex_stall = Counter('tracker_mutex_stall', '')

start_http_server(9574)

while True:
    # basic stats
    uptime.inc(stats.uptime - uptime._value.get()) # ugly
    torrents.set(stats.torrents)
    peers.set(stats.peers)
    seeds.set(stats.seeds)
    completed.inc(stats.completed - completed._value.get()) # ugly

    # connection stats
    for protocol, types in stats.connections.items():
        if protocol != 'livesync':
            for type_, count in types.items():
                connections.labels(protocol, type_).inc(count - connections.labels(protocol, type_)._value.get()) # ugly

    connections_livesync.inc(stats.connections['livesync'] - connections_livesync._value.get()) # ugly

    # “debug” stats
    for interval, count in stats.renew.items():
        renew.labels(interval).set(count)

    for code, count in stats.http_error.items():
        http_error.labels(code).inc(count - http_error.labels(code)._value.get()) # also ugly

    mutex_stall.inc(stats.mutex_stall - mutex_stall._value.get()) # ugly too

    time.sleep(refresh_interval)
    stats.update()
