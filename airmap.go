package airmap

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"
	"sync"

	"github.com/airmap/interfaces/src/go/tracking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FlightData struct {
	TrackID   string  `json:"trackId"`
	XVel      float64 `json:"xvel"`
	YVel      float64 `json:"yvel"`
	Altitude  float64 `json:"alt"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type FlightStats struct {
	FlightCnt int64   `json:"flightCount"`
	StatCnt   int64   `json:"statCount"`
	XVelAvg   float64 `json:"xvelAvg"`
	YVelAvg   float64 `json:"yvelAvg"`
	AltAvg    float64 `json:"altAvg"`
}

type Flights struct {
	processor   tracking.Collector_ConnectProcessorClient
	flightStats FlightStats
	flightMap   map[string]FlightData
	sync.RWMutex
}

func Connect() (*Flights, error) {
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

	f := Flights{}
	f.processor = stream
	f.flightMap = make(map[string]FlightData)

	return &f, err
}

func (f *Flights) Context() context.Context {
	return f.processor.Context()
}

func (f *Flights) Stream(bb *BBox, done chan bool) {
	// receive forever

	var statCnt, flightCnt int64
	var xavg, yavg, aavg float64
	flightMap := make(map[string]bool)

	for {
		update, err := f.processor.Recv()

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

			// get some flight stats
			pos := track.GetPosition().GetAbsolute()
			vel := track.GetVelocity().GetCartesian()

			lat := pos.GetCoordinate().GetLatitude().GetValue()
			lon := pos.GetCoordinate().GetLongitude().GetValue()
			alt := pos.GetAltitude().GetHeight().GetValue()

			xvel := vel.GetX().GetValue()
			yvel := vel.GetY().GetValue()

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
				fd := FlightData{}
				fd.TrackID = trackID
				fd.XVel = xvel
				fd.YVel = yvel
				fd.Altitude = alt
				fd.Latitude = lat
				fd.Longitude = lon

				if _, ok := flightMap[trackID]; !ok {
					flightMap[trackID] = true
					flightCnt++
					if flightCnt == 1 {
						log.Printf("detected first flight, %s\n", trackID)
					} else {
						log.Printf("detected %d flights, %s\n", flightCnt, trackID)
					}
				}

				f.Lock()
				f.flightMap[trackID] = fd
				f.Unlock()
			}

			statCnt++

			// get cumulative moving avg
			n := float64(statCnt)
			xavg = getCMA(math.Abs(float64(xvel)), xavg, n)
			yavg = getCMA(math.Abs(float64(yvel)), yavg, n)
			aavg = getCMA(math.Abs(float64(alt)), aavg, n)

			fs := FlightStats{}
			fs.XVelAvg = xavg
			fs.YVelAvg = yavg
			fs.AltAvg = aavg
			fs.StatCnt = statCnt
			fs.FlightCnt = flightCnt

			f.Lock()
			f.flightStats = fs
			f.Unlock()

			if statCnt%10 == 0 {
				log.Printf("Average over %d data points - x:%8.2f, y:%8.2f, alt:%8.2f\n", statCnt, xavg, yavg, aavg)
			}
		}
	}

}

// calculate cumulative moving average
func getCMA(val, avg, n float64) float64 {
	return (avg*n + val) / (n + 1.0)
}
