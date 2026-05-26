package geo

import (
	"os"
	"testing"
)

func TestGetLocationByIP_EmptyDatabasePath(t *testing.T) {
	_, err := GetLocationByIP("", "8.8.8.8")
	if err == nil {
		t.Error("GetLocationByIP should return error for empty database path")
	}
}

func TestGetLocationByIP_EmptyIP(t *testing.T) {
	_, err := GetLocationByIP("test.mmdb", "")
	if err == nil {
		t.Error("GetLocationByIP should return error for empty IP")
	}
}

func TestGetLocationByIP_InvalidIP(t *testing.T) {
	_, err := GetLocationByIP("test.mmdb", "invalid-ip")
	if err == nil {
		t.Error("GetLocationByIP should return error for invalid IP")
	}
}

func TestGetLocationByIP_NonexistentDatabase(t *testing.T) {
	_, err := GetLocationByIP("nonexistent.mmdb", "8.8.8.8")
	if err == nil {
		t.Error("GetLocationByIP should return error for nonexistent database file")
	}
}

func TestDownloadGeoLite2Database(t *testing.T) {
	tempFile := os.TempDir() + "/test_geoip.mmdb"
	defer os.Remove(tempFile)

	err := DownloadGeoLite2Database(tempFile)
	if err != nil {
		t.Skipf("Skipping test due to download error: %v", err)
	}

	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("DownloadGeoLite2Database should create the database file")
	}
}
