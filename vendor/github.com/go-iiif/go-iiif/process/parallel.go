package process

import (
	"errors"
	"fmt"
	iiifuri "github.com/go-iiif/go-iiif-uri"
	iiifconfig "github.com/go-iiif/go-iiif/config"
	iiifdriver "github.com/go-iiif/go-iiif/driver"
	iiifservice "github.com/go-iiif/go-iiif/service"
	"log"
	"net/url"
	"sync"
)

func ParallelProcessURIWithInstructionSet(cfg *iiifconfig.Config, driver iiifdriver.Driver, pr Processor, instruction_set IIIFInstructionSet, u iiifuri.URI) (map[string]interface{}, error) {

	done_ch := make(chan bool)
	err_ch := make(chan error)

	remaining := len(instruction_set) + 1 // + 1 because we fetch the palette for the source image

	results := make(map[string]interface{})

	uris := make(map[Label]string)
	dimensions := make(map[Label][]int)

	mu := new(sync.RWMutex)

	go func() {

		defer func() {
			done_ch <- true
		}()

		origin := u.Origin()

		im, err := driver.NewImageFromConfig(cfg, origin)

		if err != nil {
			msg := fmt.Sprintf("failed to derive palette for %s : %s", u, err)
			err_ch <- errors.New(msg)
			return
		}

		for _, service_name := range cfg.Profile.Services.Enable {

			if service_name == "palette" {

				s, err := iiifservice.NewPaletteService(cfg.Palette, im)

				if err != nil {
					msg := fmt.Sprintf("failed to derive palette for %s : %s", u, err)
					err_ch <- errors.New(msg)
					return
				}

				results["palette"] = s.Value()
				break
			}
		}

	}()

	for label, i := range instruction_set {

		i = EnsureInstructions(i)

		go func(u iiifuri.URI, label Label, i IIIFInstructions) {

			defer func() {
				done_ch <- true
			}()

			var process_uri iiifuri.URI

			switch u.Driver() {
			case "idsecret":

				str_label := fmt.Sprintf("%s", label)

				opts := &url.Values{}
				opts.Set("label", str_label)
				opts.Set("format", i.Format)

				if str_label == "o" {
					opts.Set("original", "1")
				}

				target_str, err := u.Target(opts)

				if err != nil {
					msg := fmt.Sprintf("failed to derive target %s (%s) : %s", u, label, err)
					err_ch <- errors.New(msg)
					return
				}

				origin := u.Origin()

				rw_str := fmt.Sprintf("%s?target=%s", origin, target_str)
				rw_str = iiifuri.NewRewriteURIString(rw_str)
				
				rw_uri, err := iiifuri.NewURI(rw_str)

				if err != nil {
					msg := fmt.Sprintf("failed to generate rewrite URL %s (%s) : %s", u, label, err)
					err_ch <- errors.New(msg)
					return
				}

				process_uri = rw_uri

			default:
				process_uri = u
			}

			new_uri, im, err := pr.ProcessURIWithInstructions(process_uri, label, i)

			if err != nil {
				msg := fmt.Sprintf("failed to process %s (%s) : %s", u.String(), label, err)
				err_ch <- errors.New(msg)
				return
			}

			dims, err := im.Dimensions()

			if err != nil {
				msg := fmt.Sprintf("failed to process %s (%s) : %s", u, label, err)
				err_ch <- errors.New(msg)
				return
			}

			mu.Lock()

			uris[label] = new_uri.String()

			dimensions[label] = []int{
				dims.Width(),
				dims.Height(),
			}

			mu.Unlock()

		}(u, label, i)
	}

	for remaining > 0 {

		select {
		case <-done_ch:
			remaining -= 1
		case e := <-err_ch:
			log.Println(e)
		default:
			//
		}
	}

	results["uris"] = uris
	results["dimensions"] = dimensions

	return results, nil
}
