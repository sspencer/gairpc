package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"

	pb "github.com/airmap/interfaces/src/go/tracking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
const bbFormat = "swlat, swlon, nelat, nelon"

type coordinate2D struct {
	lat, lon float64
}

type boundingBox struct {
	sw, ne coordinate2D
}

func main() {
	// small database of a few local spots
	bMap := make(map[string]*boundingBox)
	bMap["la"] = &boundingBox{coordinate2D{33.674069, -118.619385}, coordinate2D{34.420505, -117.993164}}
	bMap["usa"] = &boundingBox{coordinate2D{26.115986, -124.277344}, coordinate2D{49.037868, -66.269531}}
	bMap["eu"] = &boundingBox{coordinate2D{36.385913, -12.304688}, coordinate2D{71.413177, 42.626953}}

	bPtr := flag.String("b", "", bbFormat)
	flag.Parse()

	var bb *boundingBox
	var err error
	var ok bool

	loc := strings.ToLower(*bPtr)
	if bb, ok = bMap[loc]; !ok {
		bb, err = newBoundingBox(*bPtr)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	cred := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.Dial("api.airmap.com:443", grpc.WithTransportCredentials(cred))
	if err != nil {
		log.Fatalf("Failed to connect to server %v", err)
	}

	client := pb.NewCollectorClient(conn)

	stream, err := client.ConnectProcessor(context.Background())

	if err != nil {
		log.Fatalf("Failed to connect to stream: %v", err)
	}

	ctx := stream.Context()
	done := make(chan bool)

	// read from the stream
	go func() {
		var cnt int64
		var xsum, ysum, asum float64
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

				// keep running total for average
				asum += math.Abs(float64(a))
				xsum += math.Abs(float64(x))
				ysum += math.Abs(float64(y))

				cnt++

				if cnt%10 == 0 {
					log.Printf("Average over %d data points - x:%8.2f, y:%8.2f, alt:%8.2f\n",
						cnt, xsum/float64(cnt), ysum/float64(cnt), asum/float64(cnt))
				}
			}
		}
	}()

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

// parse bounding box input from CLI
func parseFloat(s, title string, min, max float64) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0.0, fmt.Errorf("Bounding Box coordinate %s: %v\n", title, err)
	}

	if f < min || f > max {
		return 0.0, fmt.Errorf("Bounding Box coordinate %s: %0.2f not within range %0.2f to %0.2f", title, f, min, max)
	}

	return f, nil
}

// create bounding box from CLI
func newBoundingBox(str string) (*boundingBox, error) {
	arr := strings.Split(str, ",")
	if len(arr) == 1 {
		if len(str) > 0 {
			return nil, fmt.Errorf("Location %q not found.  Valid locations include (eu, la, usa)", str)
		}

		return nil, nil // No Bounding Box specified (this is OK)
	} else if len(arr) != 4 {
		return nil, fmt.Errorf("Bounding Box must be specified with 4 coordinates: %q\n", bbFormat)
	}

	bb := boundingBox{}

	var swlat, swlon, nelat, nelon float64
	var err error

	if swlat, err = parseFloat(arr[0], "swlat", -90.0, 90.0); err != nil {
		return nil, err
	}

	if swlon, err = parseFloat(arr[1], "swlon", -180.0, 180.0); err != nil {
		return nil, err
	}

	if nelat, err = parseFloat(arr[2], "nelat", -90.0, 90.0); err != nil {
		return nil, err
	}

	if nelon, err = parseFloat(arr[3], "nelon", -180.0, 180.0); err != nil {
		return nil, err
	}

	bb.sw = coordinate2D{swlat, swlon}
	bb.ne = coordinate2D{nelat, nelon}

	return &bb, nil
}

func (b *boundingBox) contains(lat, lon float64) bool {
	return lat >= b.sw.lat && lat <= b.ne.lat && lon >= b.sw.lon && lon <= b.ne.lon
}
