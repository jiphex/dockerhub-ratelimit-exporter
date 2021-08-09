package rlex

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	dhrl "github.com/dafnifacility/dockerhub-ratelimit-exporter/pkg/dhubratelimit"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	flagUsername      = "username"
	flagPassword      = "password"
	flagReadPassword  = "password-stdin"
	flagInterval      = "interval"
	flagListenAddr    = "http"
	flagExtipProvider = "extip"
	flagVerbose       = "verbose"
)

//go:embed static/*
var httpfiles embed.FS

var (
	metricLimit = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dockerhub",
		Subsystem: "imagepull",
		Name:      "limit",
	}, []string{"identity"})
	metricRemaining = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dockerhub",
		Subsystem: "imagepull",
		Name:      "remaining",
	}, []string{"identity"})
	metricCheck = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dockerhub",
		Subsystem: "imagepull",
		Name:      "checkstatus",
	}, []string{"identity"})
	metricCheckedAt = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dockerhub",
		Subsystem: "imagepull",
		Name:      "checktime",
	}, []string{"identity"})
)

func getPasswordFromFlagOrStdin(cc *cli.Context) string {
	if cc.Bool(flagReadPassword) {
		pb, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Warn("unable to read password")
			return ""
		}
		return string(pb)
	}
	return cc.String(flagPassword)
}

func writeJSONErr(rw http.ResponseWriter, err error) {
	rw.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(rw, `{"error":"%s"}`, err.Error())
}

type pSummaryStatus string

const (
	pSummaryStartup = "startup"
	pSummaryError   = "error"
	pSummaryRunning = "running"
)

func setSummaryStatus(id string, as pSummaryStatus) {
	if id == "unknown" {
		// No point setting the status if auth unknown
		return
	}
	var val float64 = 0
	if as == pSummaryRunning {
		val = 1
	}
	metricCheck.WithLabelValues(id).Set(val)
}

type logMiddleware struct {
	inner http.Handler
}

func (lm *logMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.WithFields(log.Fields{
		"remote": req.RemoteAddr,
	}).Infof("%s %s", req.Method, req.URL.Path)
	lm.inner.ServeHTTP(rw, req)
}

func runHTTPExporter(cc *cli.Context) error {
	hub, err := dhrl.NewChecker(
		dhrl.WithCredentials(cc.String(flagUsername), getPasswordFromFlagOrStdin(cc)),
		dhrl.WithIPSource(cc.String(flagExtipProvider)),
	)
	if err != nil {
		return err
	}
	setSummaryStatus(hub.IdentityString(), pSummaryStartup)
	var lastres dhrl.Result
	go func() {
		ticker := time.NewTicker(cc.Duration(flagInterval))
		for {
			lastres, err = hub.Check(cc.Context)
			metricCheckedAt.WithLabelValues(hub.IdentityString()).SetToCurrentTime()
			if err != nil {
				log.WithError(err).Warn("unable to check dockerhub ratelimits")
				setSummaryStatus(hub.IdentityString(), pSummaryError)
				continue
			}
			setSummaryStatus(hub.IdentityString(), pSummaryRunning)
			metricLimit.WithLabelValues(hub.IdentityString()).Set(float64(lastres.GetLimit()))
			metricRemaining.WithLabelValues(hub.IdentityString()).Set(float64(lastres.GetRemaining()))
			<-ticker.C
		}
	}()
	hroot := mux.NewRouter()
	hroot.Use(func(h http.Handler) http.Handler {
		return &logMiddleware{
			inner: h,
		}
	})
	hroot.HandleFunc("/limit", func(rw http.ResponseWriter, req *http.Request) {
		if err != nil {
			writeJSONErr(rw, err)
			return
		}
		jb, err := json.Marshal(lastres)
		if err != nil {
			writeJSONErr(rw, err)
		}
		rw.Write(jb)
	}).Methods(http.MethodGet)
	hroot.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	sdir, _ := fs.Sub(httpfiles, "static")
	hroot.PathPrefix("/").Methods(http.MethodGet).Handler(http.FileServer(http.FS(sdir)))
	laddr := cc.String(flagListenAddr)
	log.WithField("listen-addr", laddr).Info("about to listen for HTTP")
	return http.ListenAndServe(laddr, hroot)
}

// RunMain is the main function, called from main()
func RunMain() {
	app := &cli.App{
		Name:  "dockerhub-ratelimit-exporter",
		Usage: "Shows the remaining amount of container image pulls remaining from the Docker Hub to this host",
		Flags: []cli.Flag{
			&cli.StringFlag{
				EnvVars: []string{"DOCKER_REGISTRY_USER"},
				Name:    flagUsername,
			},
			&cli.StringFlag{
				EnvVars: []string{"DOCKER_REGISTRY_PASS"},
				Name:    flagPassword,
			},
			&cli.StringFlag{
				EnvVars: []string{"LISTEN_ADDR"},
				Value:   ":55123",
				Name:    flagListenAddr,
			},
			&cli.StringFlag{
				EnvVars: []string{"EXTIP_PROVIDER"},
				Value:   "icanhazip",
				Name:    flagExtipProvider,
			},
			&cli.BoolFlag{
				EnvVars: []string{"VERBOSE"},
				Value:   false,
				Name:    flagVerbose,
				Usage:   "Enable debug log level",
				Aliases: []string{"v"},
			},
			&cli.BoolFlag{
				Name: flagReadPassword,
			},
			&cli.DurationFlag{
				Name:    flagInterval,
				Value:   300 * time.Second,
				Aliases: []string{"i"},
			},
		},
		Before: func(cc *cli.Context) error {
			// Setup logging
			if cc.Bool(flagVerbose) {
				log.SetLevel(log.DebugLevel)
			}
			return nil
		},
		Action: runHTTPExporter,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
