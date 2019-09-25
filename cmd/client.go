package main

import (
	"flag"
	"log"

	"github.com/sspencer/gairpc"
)

// expected cli input format for optional bounding box is:
//     south, west, north, east
//
// For LA:
//     go run client.go -b "33.674069,-118.619385,34.420505,-117.993164"
//
// for convenience, following locations are mapped: "eu", "la", "usa"
//     go run client.go -b la
//

func main() {
	// small database of a few local spots
	gairpc.AddBoundary("la", 33.674069, -118.619385, 34.420505, -117.993164)
	gairpc.AddBoundary("usa", 26.115986, -124.277344, 26.115986, -124.277344)
	gairpc.AddBoundary("eu", 36.385913, -12.304688, 71.413177, 42.626953)

	bPtr := flag.String("b", "", gairpc.BBoxFormat)
	flag.Parse()
	bbox, err := gairpc.NewBBox(*bPtr)
	if err != nil {
		log.Fatalf("%v", err)
	}

	stream, err := gairpc.ConnectAirMap()
	if err != nil {
		log.Fatal("%v", err)
	}

	ctx := stream.Context()
	done := make(chan bool)

	go gairpc.Stream(stream, bbox, done)

	// close done channel if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	<-done
}
