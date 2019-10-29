package main

import (
	"context"
	_ "github.com/go-iiif/go-iiif-vips"
	// "github.com/go-iiif/go-iiif/tools"
	"github.com/go-iiif/go-iiif-vips/tools"
	"log"
)

func main() {

	tool, err := tools.NewIIIFServerTool()

	if err != nil {
		log.Fatal(err)
	}

	err = tool.Run(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}