package terrain

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestCoordToTile(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100000; i++ {
		coord := MakeLatLonCoord(
			rand.Float64()*180-90,
			rand.Float64()*360-180,
		)
		zoom := uint64(rand.Int() % MaxZoom)
		pixel := coord.Pixel(zoom)
		if pixel.x < 0 || pixel.x > MapPixelSize(zoom) {
			t.Fatalf("expected limit %v-%v, got %v", 0, MapPixelSize(zoom), pixel.x)
		}
		if pixel.y < 0 || pixel.y > MapPixelSize(zoom) {
			t.Fatalf("expected limit %v-%v, got %v", 0, MapPixelSize(zoom), pixel.y)
		}
		if !pixel.Valid() {
			t.Fatalf("expected %v, got %v", true, false)
		}
		tile := pixel.Tile()
		if tile.x < 0 || tile.x > MapTileSize(zoom) {
			t.Fatalf("expected limit %v-%v, got %v", 0, MapTileSize(zoom), tile.x)
		}
		if tile.y < 0 || tile.y > MapTileSize(zoom) {
			t.Fatalf("expected limit %v-%v, got %v", 0, MapTileSize(zoom), tile.y)
		}
		if !tile.Valid() {
			t.Fatalf("expected %v, got %v", true, false)
		}
		quadKey := tile.QuadKey()
		if !quadKey.Valid() {
			t.Fatalf("expected %v, got %v", true, false)
		}
		ncoord := QuadKey([]byte(quadKey.String())).Coord()

		// make sure that pixel tiles match up
		if coord.Tile(zoom) != ncoord.Tile(zoom) {
			t.Fatalf("expected %v, got %v", coord.Tile(zoom), ncoord.Tile(zoom))
		}
	}
}
func testPanic(fn func(), expected string) (err error) {
	defer func() {
		if s := recover(); s == nil {
			err = fmt.Errorf("expected '%v', got no panic", expected)
		} else if s.(string) != expected {
			err = fmt.Errorf("expected '%v', got '%v'", expected, s.(string))
		}
	}()
	fn()
	return
}

func TestPanics(t *testing.T) {
	parts := []interface{}{
		func() { MapPixelSize(MaxZoom + 1) }, "invalid zoom",
		func() { MapTileSize(MaxZoom + 1) }, "invalid zoom",
		func() { MakeLatLonCoord(33, -115).QuadKey(MaxZoom + 1) }, "invalid zoom",
		func() { MakePixel(0, 0, MaxZoom+1) }, "invalid pixel",
		func() { MakePixel(MapPixelSize(1), 0, 1) }, "invalid pixel",
		func() { MakePixel(0, MapPixelSize(1), 1) }, "invalid pixel",
		func() { MakeZXYTile(MaxZoom+1, 0, 0) }, "invalid tile",
		func() { MakeZXYTile(1, MapTileSize(1)+1, 0) }, "invalid tile",
		func() { MakeZXYTile(1, 0, MapTileSize(1)+1) }, "invalid tile",
		func() { QuadKey("4").Coord() }, "invalid quadkey",
		func() { QuadKey(strings.Repeat("0", MaxZoom+1)).Coord() }, "invalid quadkey",
	}
	for i := 0; i < len(parts); i += 2 {
		if err := testPanic(parts[i].(func()), parts[i+1].(string)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVarious(t *testing.T) {
	tiles := MapTileSize(19) * MapTileSize(19)
	if tiles != 274877906944 {
		t.Fatalf("expected %v, got %v", 274877906944, tiles)
	}

}
