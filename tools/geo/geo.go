package geo

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type Location struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

type IPInfo struct {
	IP       string   `json:"ip"`
	Location Location `json:"location"`
}

// GeoLocator 可复用的地理位置查询器（适用于高频调用场景）
type GeoLocator struct {
	db   *geoip2.Reader
	mu   sync.RWMutex
	path string
}

// NewGeoLocator 创建地理位置查询器
func NewGeoLocator(databasePath string) (*GeoLocator, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	db, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip database: %w", err)
	}

	return &GeoLocator{db: db, path: databasePath}, nil
}

// Close 关闭数据库连接
func (g *GeoLocator) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

// Lookup 查询 IP 地理位置
func (g *GeoLocator) Lookup(ip string) (*IPInfo, error) {
	if ip == "" {
		return nil, fmt.Errorf("IP address cannot be empty")
	}
	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	return queryByLocalDatabase(g.db, ip)
}

// GetLocationByIP 通过本地 GeoIP2 数据库查询 IP 地理位置（每次打开/关闭数据库，适用于低频调用）
func GetLocationByIP(databasePath, ip string) (*IPInfo, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	if ip == "" {
		return nil, fmt.Errorf("IP address cannot be empty")
	}

	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	db, err := geoip2.Open(databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open geoip database: %w", err)
	}
	defer db.Close()

	return queryByLocalDatabase(db, ip)
}

func queryByLocalDatabase(db *geoip2.Reader, ip string) (*IPInfo, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	record, err := db.City(parsedIP)
	if err != nil {
		return nil, fmt.Errorf("failed to query geoip database: %w", err)
	}

	var countryCode string
	if len(record.Country.IsoCode) > 0 {
		countryCode = record.Country.IsoCode
	} else if len(record.RegisteredCountry.IsoCode) > 0 {
		countryCode = record.RegisteredCountry.IsoCode
	}

	var countryName string
	if len(record.Country.Names["zh-CN"]) > 0 {
		countryName = record.Country.Names["zh-CN"]
	} else if len(record.Country.Names["en"]) > 0 {
		countryName = record.Country.Names["en"]
	}

	var cityName string
	if len(record.City.Names["zh-CN"]) > 0 {
		cityName = record.City.Names["zh-CN"]
	} else if len(record.City.Names["en"]) > 0 {
		cityName = record.City.Names["en"]
	}

	var subdivisionName string
	if len(record.Subdivisions) > 0 {
		if len(record.Subdivisions[0].Names["zh-CN"]) > 0 {
			subdivisionName = record.Subdivisions[0].Names["zh-CN"]
		} else if len(record.Subdivisions[0].Names["en"]) > 0 {
			subdivisionName = record.Subdivisions[0].Names["en"]
		}
	}

	return &IPInfo{
		IP: ip,
		Location: Location{
			Country:     countryName,
			CountryCode: countryCode,
			Region:      subdivisionName,
			City:        cityName,
			Zip:         record.Postal.Code,
			Latitude:    float64(record.Location.Latitude),
			Longitude:   float64(record.Location.Longitude),
			Timezone:    record.Location.TimeZone,
		},
	}, nil
}

func DownloadGeoLite2Database(downloadPath string) error {
	if downloadPath == "" {
		downloadPath = filepath.Join(os.TempDir(), "GeoLite2-City.mmdb")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get("https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb")
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download database: status %d", resp.StatusCode)
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
