package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/radiofrance/389ds_exporter/exporter"
	log "github.com/sirupsen/logrus"
)

var (
	interval    = flag.Duration("interval", 60*time.Second, "Scrape interval")
	ipaDns      = flag.Bool("ipa-dns", true, "Should we scrape dns stats?")
	ipaDomain   = flag.String("ipa-domain", "", "FreeIPA domain e.g. example.org")
	ldapAddr    = flag.String("ldap.addr", "localhost:389", "Address of 389ds server")
	ldapUser    = flag.String("ldap.user", "cn=Directory Manager", "389ds Directory Manager user")
	ldapPass    = flag.String("ldap.pass", "", "389ds Directory Manager password")
	logLevel    = flag.String("log.level", "info", "Log level")
	logFormat   = flag.String("log.format", "default", "Log format (default, json)")
	listenPort  = flag.String("web.listen-address", ":9496", "Bind address for prometheus HTTP metrics server")
	metricsPath = flag.String("web.telemetry-path", "/metrics", "Path to expose metrics on")
	showVersion = flag.Bool("version", false, "Exporter version")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Print("Version"))
		return
	}

	if level, err := log.ParseLevel(*logLevel); err != nil {
		log.Fatalf("log.level must be one of %v", log.AllLevels)
	} else {
		log.SetLevel(level)
	}

	switch *logFormat {
	case "default":
		log.SetFormatter(&log.TextFormatter{})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.Fatal("log.level must be one of [default json]")
	}

	if *ldapPass == "" {
		log.Fatal("ldap.pass cannot be empty")
	}
	if *ipaDomain == "" {
		log.Fatal("ipa-domain cannot be empty")
	}

	log.Info("Starting prometheus HTTP metrics server on ", *listenPort)
	go StartMetricsServer(*listenPort)

	log.Info("Starting 389ds scraper for ", *ldapAddr)
	for range time.Tick(*interval) {
		log.Debug("Starting metrics scrape")
		exporter.ScrapeMetrics(*ldapAddr, *ldapUser, *ldapPass, *ipaDomain, *ipaDns)
	}
}

func StartMetricsServer(bindAddr string) {
	d := http.NewServeMux()
	d.Handle(*metricsPath, promhttp.Handler())
	d.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>389ds Exporter</title></head>
             <body>
             <h1>389ds Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </dl>
             <h2>Build</h2>
             <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
             </body>
             </html>`))
	})

	err := http.ListenAndServe(bindAddr, d)
	if err != nil {
		log.Fatal("Failed to start metrics server, error is:", err)
	}
}
