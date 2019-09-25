package airmap

import (
	"fmt"
	"strconv"
	"strings"
)

const BBoxFormat = "swlat, swlon, nelat, nelon"

var bbMap map[string]*BBox

type Coordinate2D struct {
	lat, lon float64
}

type BBox struct {
	sw, ne Coordinate2D
}

func init() {
	bbMap = make(map[string]*BBox)
}

func AddBoundary(name string, swlat, swlon, nelat, nelon float64) {
	bbMap[name] = &BBox{Coordinate2D{swlat, swlon}, Coordinate2D{nelat, nelon}}
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
func NewBBox(str string) (*BBox, error) {
	arr := strings.Split(str, ",")
	if len(arr) == 1 {
		str = strings.ToLower(strings.TrimSpace(str))
		if len(str) == 0 {
			return nil, nil // no BBox specified
		}

		if bb, ok := bbMap[str]; !ok {
			return nil, fmt.Errorf("Location %q not found.", str)
		} else {
			return bb, nil
		}
	} else if len(arr) != 4 {
		return nil, fmt.Errorf("Bounding Box must be specified with 4 coordinates: %q\n", BBoxFormat)
	}

	bb := BBox{}

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

	bb.sw = Coordinate2D{swlat, swlon}
	bb.ne = Coordinate2D{nelat, nelon}

	return &bb, nil
}

func (b *BBox) contains(lat, lon float64) bool {
	return lat >= b.sw.lat && lat <= b.ne.lat && lon >= b.sw.lon && lon <= b.ne.lon
}
