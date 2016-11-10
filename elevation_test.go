package terrain

import (
	"fmt"
	"testing"
)

func TestElevation(t *testing.T) {
	//lat, lon := 27.9878, 86.9250 // everest
	lat, lon := 31.5590, 35.4732
	//lat,lon:=33.5, -114.5 // arizona
	s := NewElevationService(nil)
	tlx, tly, zoom := MakeLatLonCoord(lat, lon).Tile(MaxElevationZoom - 10).Pixel().XYZoom()
	for y := tly; y < tly+256; y++ {
		for x := tlx; x < tlx+256; x++ {
			meters, err := s.AtPixel(MakePixel(x, y, zoom))
			if err != nil {
				t.Fatal(err)
			}
			fmt.Printf("%v\n", meters)
		}
	}
	return
	for i := 0; i < 256*256; i++ {

		meters, err := s.AtCoordZoom(MakeLatLonCoord(lat, lon), 5)
		if err != nil {
			t.Fatal(err)
		}
		continue
		fmt.Printf("%v\n", meters)
	}
}
