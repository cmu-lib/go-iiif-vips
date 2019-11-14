package tools

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/aaronland/gocloud-blob-bucket"
	iiifcache "github.com/go-iiif/go-iiif/cache"
	iiifconfig "github.com/go-iiif/go-iiif/config"
	iiifdriver "github.com/go-iiif/go-iiif/driver"
	iiifhttp "github.com/go-iiif/go-iiif/http"
	iiiflevel "github.com/go-iiif/go-iiif/level"
	iiifserver "github.com/go-iiif/go-iiif/server"
	iiifsource "github.com/go-iiif/go-iiif/source"
	"github.com/gorilla/mux"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

type IIIFServerTool struct {
	Tool
}

func NewIIIFServerTool() (Tool, error) {

	t := &IIIFServerTool{}
	return t, nil
}

func (t *IIIFServerTool) Run(ctx context.Context) error {

	var cfg = flag.String("config", "", "Path to a valid go-iiif config file. DEPRECATED - please use -config-url and -config name.")

	var config_source = flag.String("config-source", "", "A valid Go Cloud bucket URI where your go-iiif config file is located.")
	var config_name = flag.String("config-name", "config.json", "The name of your go-iiif config file.")

	var proto = flag.String("protocol", "http", "The protocol for wof-staticd server to listen on. Valid protocols are: http, lambda.")
	var host = flag.String("host", "localhost", "Bind the server to this host")
	var port = flag.Int("port", 8080, "Bind the server to this port")
	var example = flag.Bool("example", false, "Add an /example endpoint to the server for testing and demonstration purposes")
	var root = flag.String("example-root", "example", "An explicit path to a folder containing example assets")

	flag.Parse()

	if *cfg != "" {

		log.Println("-config flag is deprecated. Please use -config-source and -config-name (setting them now).")

		abs_config, err := filepath.Abs(*cfg)

		if err != nil {
			return err
		}

		*config_name = filepath.Base(abs_config)
		*config_source = fmt.Sprintf("file://%s", filepath.Dir(abs_config))
	}

	if *config_source == "" {
		return errors.New("Required -config-source flag is empty.")
	}

	config_bucket, err := bucket.OpenBucket(ctx, *config_source)

	if err != nil {
		return err
	}

	config, err := iiifconfig.NewConfigFromBucket(ctx, config_bucket, *config_name)

	if err != nil {
		return err
	}

	driver, err := iiifdriver.NewDriverFromConfig(config)

	if err != nil {
		return err
	}

	/*
		See this - we're just going to make sure we have a valid source
		before we start serving images (20160901/thisisaaronland)
	*/

	_, err = iiifsource.NewSourceFromConfig(config)

	if err != nil {
		return err
	}

	_, err = iiiflevel.NewLevelFromConfig(config, *host)

	if err != nil {
		return err
	}

	/*

		Okay now we're going to set up global cache thingies for source images
		and derivatives mostly to account for the fact that in-memory cache
		thingies need to be... well, global

	*/

	images_cache, err := iiifcache.NewImagesCacheFromConfig(config)

	if err != nil {
		return err
	}

	derivatives_cache, err := iiifcache.NewDerivativesCacheFromConfig(config)

	if err != nil {
		return err
	}

	info_handler, err := iiifhttp.InfoHandler(config, driver)

	if err != nil {
		return err
	}

	image_handler, err := iiifhttp.ImageHandler(config, driver, images_cache, derivatives_cache)

	if err != nil {
		return err
	}

	ping_handler, err := iiifhttp.PingHandler()

	if err != nil {
		return err
	}

	expvar_handler, err := iiifhttp.ExpvarHandler(*host)

	if err != nil {
		return err
	}

	router := mux.NewRouter()

	router.HandleFunc("/ping", ping_handler)
	router.HandleFunc("/debug/vars", expvar_handler)

	// https://github.com/go-iiif/go-iiif/issues/4

	router.HandleFunc("/{identifier:.+}/info.json", info_handler)
	router.HandleFunc("/{identifier:.+}/{region}/{size}/{rotation}/{quality}.{format}", image_handler)

	if *example {

		abs_path, err := filepath.Abs(*root)

		if err != nil {
			return err
		}

		_, err = os.Stat(abs_path)

		if os.IsNotExist(err) {
			return err
		}

		example_handler, err := iiifhttp.ExampleHandler(abs_path)

		if err != nil {
			return err
		}

		router.HandleFunc("/example/{ignore:.*}", example_handler)
	}

	address := fmt.Sprintf("http://%s:%d", *host, *port)

	u, err := url.Parse(address)

	if err != nil {
		return err
	}

	s, err := iiifserver.NewServer(*proto, u)

	if err != nil {
		return err
	}

	log.Printf("Listening on %s\n", s.Address())

	err = s.ListenAndServe(router)

	if err != nil {
		return err
	}

	return nil
}
