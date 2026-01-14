package grpc

import (
	"monitor/pkg/ipgeo"
)

type IPGeoService struct {
	service *ipgeo.Service
}

func NewIPGeoService() *IPGeoService {
	return &IPGeoService{
		service: ipgeo.NewService(),
	}
}

func (s *IPGeoService) QueryIP(ip string) (*ipgeo.IPGeoResponse, error) {
	return s.service.QueryIP(ip)
}