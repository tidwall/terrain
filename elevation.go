package terrain

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const MaxElevationZoom = 14

type cacheKey struct {
	z, x, y uint64
}

type ElevationService struct {
	mu    sync.RWMutex
	cache map[cacheKey][]byte
	lru   []cacheKey
	lrui  int
}
type ElevationServiceOptions struct {
}

func NewElevationService(opts *ElevationServiceOptions) *ElevationService {
	return &ElevationService{
		cache: make(map[cacheKey][]byte),
		lru:   make([]cacheKey, 1024),
	}
}

func (s *ElevationService) AtCoord(coord Coord) (meters float64, err error) {
	return s.AtCoordZoom(coord, MaxElevationZoom)
}

func (s *ElevationService) AtCoordZoom(coord Coord, zoom uint64) (meters float64, err error) {
	return s.AtPixel(coord.Pixel(zoom))
}

func (s *ElevationService) AtPixel(pixel Pixel) (meters float64, err error) {
	ptile, err := s.PixelTile(pixel.Tile())
	if err != nil {
		return 0, err
	}
	xx := pixel.x % tileSize
	yy := pixel.y % tileSize
	idx := yy*tileSize*4 + xx*4
	red := float64(ptile.Pix[idx+0])
	green := float64(ptile.Pix[idx+1])
	blue := float64(ptile.Pix[idx+2])
	meters = (red*256 + green + blue/256) - 32768
	return
}

type PixelTile struct {
	Tile Tile
	Pix  []byte
}

func (s *ElevationService) PixelTile(tile Tile) (PixelTile, error) {
	pix, err := s.getTilePix(tile.ZXY())
	if err != nil {
		return PixelTile{}, err
	}
	return PixelTile{Tile: tile, Pix: pix}, nil
}

func (s *ElevationService) getTilePix(z, x, y uint64) ([]byte, error) {
	if z > MaxElevationZoom {
		return nil, fmt.Errorf("zoom level cannot be greater than %d", MaxElevationZoom)
	}
	s.mu.RLock()
	pix, ok := s.cache[cacheKey{z, x, y}]
	s.mu.RUnlock()
	if ok {
		return pix, nil
	}
	var err error
	pix, err = wgetTerrarium(z, x, y)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lrui >= len(s.lru) {
		delete(s.cache, s.lru[s.lrui%len(s.lru)])
	}
	s.lru[s.lrui%len(s.lru)] = cacheKey{z, x, y}
	s.lrui++
	s.cache[cacheKey{z, x, y}] = pix
	return pix, nil
}

const terrariumURL = "https://tile.mapzen.com/mapzen/terrain/v1/terrarium/{z}/{x}/{y}.png"

func makeURL(url string, z, x, y uint64) string {
	url = strings.Replace(url, "{z}", strconv.FormatUint(z, 10), -1)
	url = strings.Replace(url, "{x}", strconv.FormatUint(x, 10), -1)
	url = strings.Replace(url, "{y}", strconv.FormatUint(y, 10), -1)
	return url
}

func wget(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func wgetTerrarium(z, x, y uint64) ([]byte, error) {
	data, err := wget(makeURL(terrariumURL, z, x, y))
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	if img.Bounds().Size().X != 256 || img.Bounds().Size().Y != 256 {
		return nil, errors.New("invalid png, expecting 256x256 pixel image")
	}
	piximg := image.NewNRGBA(img.Bounds())
	draw.Draw(piximg, img.Bounds(), img, image.ZP, draw.Src)
	if piximg.Stride != 1024 {
		return nil, errors.New("invalid pix, expecting 1024 pixel stride")
	}
	return piximg.Pix, nil
}
