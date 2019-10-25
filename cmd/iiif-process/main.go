package main

import (
	"context"
	_ "github.com/go-iiif/go-iiif-vips"
	"github.com/go-iiif/go-iiif-vips/tools"
	// "github.com/go-iiif/go-iiif/tools"
	"log"
)

func main() {

	tool, err := tools.NewProcessTool()

	if err != nil {
		log.Fatal(err)
	}

	err = tool.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
