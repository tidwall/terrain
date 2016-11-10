package terrain

import "math"

const (
	earthRadius = 6378137.0 // meters
	originShift = 2 * math.Pi * earthRadius / 2.0
	tileSize    = 256                // pixels
	minLat      = -85.05112877980659 // min y merc
	maxLat      = 85.05112877980659  // max y merv
	minLon      = -180.0             // min x merc
	maxLon      = 180.0              // max x merc
	// MaxZoom represents that maximum zoom level for safe calculations.
	MaxZoom = 38
)

func MapPixelSize(zoom uint64) uint64 {
	if zoom > MaxZoom {
		panic("invalid zoom")
	}
	return tileSize << zoom
}

func MapTileSize(zoom uint64) uint64 {
	if zoom > MaxZoom {
		panic("invalid zoom")
	}
	return tileSize << zoom >> 8
}

func clip(n, min, max float64) float64 {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

type Pixel struct {
	x, y, zoom uint64
}

func (pixel Pixel) XYZoom() (x, y, zoom uint64) {
	return pixel.x, pixel.y, pixel.zoom
}

func MakePixel(x, y, zoom uint64) Pixel {
	pixel := Pixel{x, y, zoom}
	if !pixel.Valid() {
		panic("invalid pixel")
	}
	return pixel
}

func (pixel Pixel) Valid() bool {
	if pixel.zoom > MaxZoom {
		return false
	}
	return pixel.x < tileSize<<pixel.zoom && pixel.y < tileSize<<pixel.zoom
}

func (pixel Pixel) Coord() Coord {
	if !pixel.Valid() {
		panic("invalid pixel")
	}
	mapSize := float64(MapPixelSize(pixel.zoom))
	x := (clip(float64(pixel.x), 0, mapSize-1) / mapSize) - 0.5
	y := 0.5 - (clip(float64(pixel.y), 0, mapSize-1) / mapSize)
	return MakeLatLonCoord(
		90-360*math.Atan(math.Exp(-y*2*math.Pi))/math.Pi,
		360*x,
	)
}

func (pixel Pixel) Tile() Tile {
	if !pixel.Valid() {
		panic("invalid pixel")
	}
	return Tile{
		x: pixel.x >> 8,
		y: pixel.y >> 8,
		z: pixel.zoom,
	}
}

func (pixel Pixel) QuadKey() QuadKey {
	return pixel.Tile().QuadKey()
}

type Coord struct {
	x, y  float64
	datum Datum
}

func MakeLatLonCoord(lat, lon float64) Coord {
	coord := Coord{x: lon, y: lat, datum: EPSG4326}
	if !coord.Valid() {
		panic("invalid coord")
	}
	return coord
}

func MakeMetersCoord(x, y float64) Coord {
	coord := Coord{x: x, y: y, datum: EPSG3857}
	if !coord.Valid() {
		panic("invalid coord")
	}
	return coord
}

func (coord Coord) Datum() Datum {
	return coord.datum
}
func (coord Coord) Valid() bool {
	switch coord.datum {
	default:
		return false
	case EPSG3857:
	case EPSG4326:
	}
	return true
}

func (coord Coord) LatLon() (lat, lon float64) {
	switch coord.datum {
	case EPSG3857:
		lat, lon = epsg3857To4326(coord.x, coord.y)
	case EPSG4326:
		lat, lon = coord.y, coord.x
	}
	return
}

func (coord Coord) Meters() (x, y float64) {
	switch coord.datum {
	case EPSG3857:
		x, y = coord.x, coord.y
	case EPSG4326:
		x, y = epsg4326To3857(coord.y, coord.x)
	}
	return
}

/*
// GroundResolution returns the ground resolution, in meters per pixel.
func (coord Coord) GroundResolution(zoom uint64) float64 {
	lat := clip(coord.Y, minLat, maxLat)
	return math.Cos(lat*math.Pi/180) * 2 * math.Pi * earthRadius / float64(MapPixelSize(zoom))
}
*/
func (coord Coord) Pixel(zoom uint64) Pixel {
	if zoom > MaxZoom {
		panic("invalid zoom")
	}
	lat, lon := coord.LatLon()
	lat = clip(lat, minLat, maxLat)
	lon = clip(lon, minLon, maxLon)
	xx := (lon + 180) / 360
	sinLat := math.Sin(lat * math.Pi / 180)
	yy := 0.5 - math.Log((1+sinLat)/(1-sinLat))/(4*math.Pi)
	mapSize := float64(MapPixelSize(zoom))
	return MakePixel(
		uint64(clip(xx*mapSize+0.5, 0, mapSize-1)),
		uint64(clip(yy*mapSize+0.5, 0, mapSize-1)),
		uint64(zoom),
	)
}

func (coord Coord) Tile(zoom uint64) Tile {
	return coord.Pixel(zoom).Tile()
}

func (coord Coord) QuadKey(zoom uint64) QuadKey {
	return coord.Pixel(zoom).QuadKey()
}

type Tile struct {
	z, x, y uint64
}

func MakeZXYTile(z, x, y uint64) Tile {
	tile := Tile{z: z, x: x, y: y}
	if !tile.Valid() {
		panic("invalid tile")
	}
	return tile
}
func (tile Tile) ZXY() (z, x, y uint64) {
	return tile.z, tile.x, tile.y
}
func (tile Tile) Valid() bool {
	if tile.z > MaxZoom {
		return false
	}
	return tile.x < tileSize<<tile.z>>8 && tile.y < tileSize<<tile.z>>8
}

func (tile Tile) Coord() Coord {
	return tile.Pixel().Coord()
}

func (tile Tile) Pixel() Pixel {
	return MakePixel(
		tile.x<<8,
		tile.y<<8,
		tile.z,
	)
}

func (tile Tile) QuadKey() QuadKey {
	quadKey := make([]byte, tile.z)
	for i, j := tile.z, 0; i > 0; i, j = i-1, j+1 {
		mask := uint64(1 << (i - 1))
		if (tile.x & mask) != 0 {
			if (tile.y & mask) != 0 {
				quadKey[j] = '3'
			} else {
				quadKey[j] = '1'
			}
		} else if (tile.y & mask) != 0 {
			quadKey[j] = '2'
		} else {
			quadKey[j] = '0'
		}
	}
	return QuadKey(quadKey)
}

func (tile Tile) Bounds() (min, max Coord) {
	z, x, y := tile.ZXY()
	mapSize := MapPixelSize(z)
	nwLat, nwLon := tile.Coord().LatLon()
	var seLat, seLon float64
	if x+1 == mapSize {
		if y+1 == mapSize {
			seLat = maxLat
			seLon = maxLon
		} else {
			seLat, _ = MakeZXYTile(z, x, y).Coord().LatLon()
			seLon = maxLon
		}
	} else {
		if y+1 == mapSize {
			seLat = maxLat
			_, seLon = MakeZXYTile(z, x+1, y).Coord().LatLon()
		} else {
			seLat, seLon = MakeZXYTile(z, x+1, y+1).Coord().LatLon()
		}
	}
	min = MakeLatLonCoord(seLat, nwLon)
	max = MakeLatLonCoord(nwLat, seLon)
	return
}

type QuadKey []byte

func (quadKey QuadKey) String() string {
	return string([]byte(quadKey))
}

func (quadKey QuadKey) Valid() bool {
	if len(quadKey) > MaxZoom {
		return false
	}
	for i := 0; i < len(quadKey); i++ {
		if quadKey[i] < '0' || quadKey[i] > '3' {
			return false
		}
	}
	return true
}

func (quadKey QuadKey) Coord() Coord {
	return quadKey.Pixel().Coord()
}

func (quadKey QuadKey) Pixel() Pixel {
	return quadKey.Tile().Pixel()
}

func (quadKey QuadKey) Tile() Tile {
	if !quadKey.Valid() {
		panic("invalid quadkey")
	}
	var tile Tile
	tile.z = uint64(len(quadKey))
	for i := len(quadKey); i > 0; i-- {
		mask := uint64(1 << (byte(i) - 1))
		switch quadKey[len(quadKey)-i] {
		case '0':
		case '1':
			tile.x |= mask
		case '2':
			tile.y |= mask
		case '3':
			tile.x |= mask
			tile.y |= mask
		}
	}
	return tile
}

type Meters struct {
	X, Y float64
}

type Datum int

const (
	EPSG3857 Datum = 3857
	EPSG4326 Datum = 4326
)

//Converts XY point from Spherical (EPSG:3857) Mercator EPSG:900913 to lat/lon in WGS84 Datum
func epsg3857To4326(x, y float64) (lat, lon float64) {
	lon = (x / originShift) * 180.0
	lat = (y / originShift) * 180.0
	lat = 180 / math.Pi * (2*math.Atan(math.Exp(lat*math.Pi/180.0)) - math.Pi/2.0)
	return
}

func epsg4326To3857(lat, lon float64) (x, y float64) {
	x = lon * originShift / 180.0
	y = math.Log(math.Tan((90+lat)*math.Pi/360.0)) / (math.Pi / 180.0)
	y = y * originShift / 180.0
	return
}
