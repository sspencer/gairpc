package main

import (
	"flag"
	"log"

	"github.com/sspencer/airmap"
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
	airmap.AddBoundary("la", 33.674069, -118.619385, 34.420505, -117.993164)
	airmap.AddBoundary("usa", 26.115986, -124.277344, 26.115986, -124.277344)
	airmap.AddBoundary("eu", 36.385913, -12.304688, 71.413177, 42.626953)

	bPtr := flag.String("b", "", airmap.BBoxFormat)
	flag.Parse()
	bbox, err := airmap.NewBBox(*bPtr)
	if err != nil {
		log.Fatalf("%v", err)
	}

	flights, err := airmap.Connect()
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx := flights.Context()
	done := make(chan bool)

	go flights.Stream(bbox, done)
	go flights.Redislike(":6060")

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
