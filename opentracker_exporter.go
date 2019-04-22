package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// basic stats
	trackerUptime = prometheus.NewDesc(
		"tracker_uptime_total",
		"Second the tracker has been up",
		nil, nil,
	)
	trackerTorrents = prometheus.NewDesc(
		"tracker_torrents",
		"Number of tracked torrents",
		nil, nil,
	)
	trackerPeers = prometheus.NewDesc(
		"tracker_peers",
		"Number of known peers",
		nil, nil,
	)
	trackerSeeds = prometheus.NewDesc(
		"tracker_seeds",
		"Number of known seeds",
		nil, nil,
	)
	trackerCompleted = prometheus.NewDesc(
		"tracker_completed_total",
		"Number of completed torrents",
		nil, nil,
	)
	trackerMutexStall = prometheus.NewDesc(
		"tracker_mutex_stall_total",
		"",
		nil, nil,
	)

	// connection stats
	trackerConnections = prometheus.NewDesc(
		"tracker_connections_total",
		"Number of clients renewing the connection at a specific interval",
		[]string{"protocol", "type"}, nil,
	)
	trackerConnectionsLivesync = prometheus.NewDesc(
		"tracker_connections_livesync_total",
		"Number of livesync connections",
		nil, nil,
	)

	// debug stats
	trackerRenew = prometheus.NewDesc(
		"tracker_renew_total",
		"Number of renews at a specific interval",
		[]string{"interval"}, nil,
	)
	trackerHTTPError = prometheus.NewDesc(
		"tracker_http_error_total",
		"Number of http errors",
		[]string{"code"}, nil,
	)
)

type TCPConnections struct {
	Accept   float64 `xml:"accept"`
	Announce float64 `xml:"announce"`
	Scrape   float64 `xml:"scrape"`
}

type UDPConnections struct {
	Overall   float64 `xml:"overall"`
	Connect   float64 `xml:"connect"`
	Announce  float64 `xml:"announce"`
	Scrape    float64 `xml:"scrape"`
	Missmatch float64 `xml:"missmatch"`
}

type Connections struct {
	TCP      TCPConnections `xml:"tcp"`
	UDP      UDPConnections `xml:"udp"`
	Livesync float64        `xml:"livesync>count"`
}

type Renew struct {
	Interval int     `xml:"interval,attr"`
	Count    float64 `xml:",chardata"`
}

type HTTPError struct {
	Code  string  `xml:"code,attr"`
	Count float64 `xml:",chardata"`
}

type Stats struct {
	Uptime      float64     `xml:"uptime"`
	Torrents    float64     `xml:"torrents>count_mutex"`
	Peers       float64     `xml:"peers>count"`
	Seeds       float64     `xml:"seeds>count"`
	Completed   float64     `xml:"completed>count"`
	Connections Connections `xml:"connections"`
	Renew       []Renew     `xml:"debug>renew>count"`
	HTTPErrors  []HTTPError `xml:"debug>http_error>count"`
	MutexStall  float64     `xml:"debug>mutex_stall>count"`
}

type Exporter struct {
	URL string
}

func (e Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- trackerUptime
	ch <- trackerTorrents
	ch <- trackerPeers
	ch <- trackerSeeds
	ch <- trackerCompleted
	ch <- trackerMutexStall

	ch <- trackerConnections
	ch <- trackerConnectionsLivesync

	ch <- trackerRenew
	ch <- trackerHTTPError
}

func (e Exporter) Collect(ch chan<- prometheus.Metric) {
	resp, err := http.Get(fmt.Sprintf("http://%s/stats?mode=everything", e.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	stats := Stats{}
	err = xml.Unmarshal(body, &stats)
	if err != nil {
		log.Fatal(err)
	}

	// basic stats
	ch <- prometheus.MustNewConstMetric(trackerUptime, prometheus.CounterValue, stats.Uptime)
	ch <- prometheus.MustNewConstMetric(trackerTorrents, prometheus.GaugeValue, stats.Torrents)
	ch <- prometheus.MustNewConstMetric(trackerPeers, prometheus.GaugeValue, stats.Peers)
	ch <- prometheus.MustNewConstMetric(trackerSeeds, prometheus.GaugeValue, stats.Seeds)
	ch <- prometheus.MustNewConstMetric(trackerCompleted, prometheus.CounterValue, stats.Completed)
	ch <- prometheus.MustNewConstMetric(trackerMutexStall, prometheus.CounterValue, stats.MutexStall)

	// connection stats
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.TCP.Accept, "tcp", "accept")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.TCP.Announce, "tcp", "announce")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.TCP.Scrape, "tcp", "scrape")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.UDP.Overall, "udp", "overall")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.UDP.Connect, "udp", "connect")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.UDP.Announce, "udp", "announce")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.UDP.Scrape, "udp", "scrape")
	ch <- prometheus.MustNewConstMetric(trackerConnections, prometheus.CounterValue, stats.Connections.UDP.Missmatch, "udp", "missmatch")
	ch <- prometheus.MustNewConstMetric(trackerConnectionsLivesync, prometheus.CounterValue, stats.Connections.Livesync)

	// debug stats
	for _, renew := range stats.Renew {
		ch <- prometheus.MustNewConstMetric(trackerRenew, prometheus.CounterValue, renew.Count, strconv.Itoa(renew.Interval))
	}

	for _, httpError := range stats.HTTPErrors {
		ch <- prometheus.MustNewConstMetric(trackerHTTPError, prometheus.CounterValue, httpError.Count, httpError.Code)
	}
}

func main() {
	e := Exporter{URL: os.Getenv("OPENTRACKER_URL")}
	if e.URL == "" {
		log.Fatal("Please specify the environment variable OPENTRACKER_URL")
	}
	prometheus.MustRegister(e)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9574", nil))
}
