package ipgeo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"monitor/internal/database"
	"monitor/internal/models"
)

type IPGeoResponse struct {
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	ISP       string  `json:"isp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Service struct {
	httpClient *http.Client
	apiURL     string
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiURL: "http://ip-api.com/json/",
	}
}

func (s *Service) QueryIP(ip string) (*IPGeoResponse, error) {
	db := database.GetDB()

	var cached models.IPGeoCache
	if err := db.Where("ip = ?", ip).First(&cached).Error; err == nil {
		return &IPGeoResponse{
			IP:        cached.IP,
			Country:   cached.Country,
			Region:    cached.Region,
			City:      cached.City,
			ISP:       cached.ISP,
			Latitude:  cached.Latitude,
			Longitude: cached.Longitude,
		}, nil
	}

	resp, err := s.queryAPI(ip)
	if err != nil {
		return nil, err
	}

	cached = models.IPGeoCache{
		IP:        resp.IP,
		Country:   resp.Country,
		Region:    resp.Region,
		City:      resp.City,
		ISP:       resp.ISP,
		Latitude:  resp.Latitude,
		Longitude: resp.Longitude,
	}

	if err := db.Create(&cached).Error; err != nil {
		if err := db.Save(&cached).Error; err != nil {
			return nil, fmt.Errorf("failed to cache IP geo data: %w", err)
		}
	}

	return resp, nil
}

func (s *Service) queryAPI(ip string) (*IPGeoResponse, error) {
	u, err := url.Parse(s.apiURL + ip)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL: %w", err)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result IPGeoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}