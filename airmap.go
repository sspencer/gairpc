package airmap

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/airmap/interfaces/src/go/tracking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Connect() (tracking.Collector_ConnectProcessorClient, error) {
	var err error

	cred := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.Dial("api.airmap.com:443", grpc.WithTransportCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to server %v", err)
	}

	client := tracking.NewCollectorClient(conn)

	stream, err := client.ConnectProcessor(context.Background())

	if err != nil {
		return nil, fmt.Errorf("Failed to connect to stream: %v", err)
	}

	return stream, err
}

func Stream(stream tracking.Collector_ConnectProcessorClient, bb *BBox, done chan bool) {
	var cnt int64
	var xavg, yavg, aavg float64

	flightMap := make(map[string]bool)
	flightCnt := 0

	// receive forever
	for {
		update, err := stream.Recv()

		if err == io.EOF {
			close(done)
			return
		}

		if err != nil {
			log.Fatalf("%v", err)
			continue
		}

		// 1 or more Tracks
		batch := update.GetBatch()
		for _, track := range batch.Tracks {

			// when bounding box is set, filter flights not contained with BB
			if bb != nil {
				lat := track.GetPosition().GetAbsolute().GetCoordinate().GetLatitude().GetValue()
				lon := track.GetPosition().GetAbsolute().GetCoordinate().GetLongitude().GetValue()

				if !bb.contains(lat, lon) {
					continue
				}
			}

			// flight can be identified in more than 1 way - just look for TrackID
			ids := track.GetIdentities()
			trackID := ""
			for _, id := range ids {
				trackID = id.GetTrackId().GetAsString()
				if len(trackID) > 0 {
					break
				}
			}

			// if a trackID was found, perform Map lookup to see if it's unique
			if len(trackID) > 0 {
				if _, ok := flightMap[trackID]; !ok {
					flightMap[trackID] = true
					flightCnt++
					if flightCnt == 1 {
						log.Println("detected first flight")
					} else {
						log.Printf("detected %d flights\n", flightCnt)
					}
				}
			}

			// get some flight stats
			a := track.GetPosition().GetAbsolute().GetAltitude().GetHeight().GetValue()
			x := track.GetVelocity().GetCartesian().GetX().GetValue()
			y := track.GetVelocity().GetCartesian().GetY().GetValue()

			cnt++

			// get cumulative moving avg
			n := float64(cnt)
			xavg = getCMA(math.Abs(float64(x)), xavg, n)
			yavg = getCMA(math.Abs(float64(y)), yavg, n)
			aavg = getCMA(math.Abs(float64(a)), aavg, n)

			if cnt%10 == 0 {
				log.Printf("Average over %d data points - x:%8.2f, y:%8.2f, alt:%8.2f\n", cnt, xavg, yavg, aavg)
			}
		}
	}

}

// calculate cumulative moving average
func getCMA(val, avg, n float64) float64 {
	return (avg*n + val) / (n + 1.0)
}
