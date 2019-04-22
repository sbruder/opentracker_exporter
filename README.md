opentracker Exporter for Prometheus
===================================

This exporter exports statistics from an
[opentracker](https://erdgeist.org/arts/software/opentracker/) instance to
Prometheus via the [`/stats`
path](https://erdgeist.org/arts/software/opentracker/#statistics).

Usage
-----

```
docker run -d --name opentracker_exporter -p 9574:9574 -e OPENTRACKER_URL=tracker.example.com:6969 sbruder/opentracker_exporter
```

Replace tracker.example.com:6969 with the host and port of your tracker.

Metrics are available on http://localhost:9574/metrics
