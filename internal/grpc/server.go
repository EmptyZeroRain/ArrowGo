package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"monitor/internal/database"
	"monitor/internal/models"
	"monitor/internal/monitor"
	pb "monitor/proto"

	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedMonitorServiceServer
	pb.UnimplementedIPGeoServiceServer
	monitorService *monitor.Service
}

func NewServer(monitorService *monitor.Service) *Server {
	return &Server{
		monitorService: monitorService,
	}
}

func (s *Server) AddMonitor(ctx context.Context, req *pb.Target) (*pb.MonitorResponse, error) {
	db := database.GetDB()

	var metadata string
	if len(req.Metadata) > 0 {
		bytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return &pb.MonitorResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to marshal metadata: %v", err),
			}, nil
		}
		metadata = string(bytes)
	}

	target := models.MonitorTarget{
		Name:     req.Name,
		Type:     req.Type,
		Address:  req.Address,
		Port:     req.Port,
		Interval: req.Interval,
		Metadata: metadata,
		Enabled:  req.Enabled,
	}

	if err := db.Create(&target).Error; err != nil {
		return &pb.MonitorResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create target: %v", err),
		}, nil
	}

	monitorTarget := &monitor.MonitorTarget{
		ID:       target.ID,
		Name:     target.Name,
		Type:     target.Type,
		Address:  target.Address,
		Port:     target.Port,
		Interval: target.Interval,
		Metadata: req.Metadata,
		Enabled:  target.Enabled,
	}

	if err := s.monitorService.AddTarget(monitorTarget); err != nil {
		return &pb.MonitorResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to add monitor: %v", err),
		}, nil
	}

	return &pb.MonitorResponse{
		Success: true,
		Message: fmt.Sprintf("Monitor added with ID: %d", target.ID),
	}, nil
}

func (s *Server) RemoveMonitor(ctx context.Context, req *pb.MonitorID) (*pb.MonitorResponse, error) {
	db := database.GetDB()

	if err := db.Delete(&models.MonitorTarget{}, req.Id).Error; err != nil {
		return &pb.MonitorResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete target: %v", err),
		}, nil
	}

	if err := s.monitorService.RemoveTarget(req.Id); err != nil {
		return &pb.MonitorResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to remove monitor: %v", err),
		}, nil
	}

	return &pb.MonitorResponse{
		Success: true,
		Message: "Monitor removed successfully",
	}, nil
}

func (s *Server) GetMonitor(ctx context.Context, req *pb.MonitorID) (*pb.Target, error) {
	db := database.GetDB()

	var target models.MonitorTarget
	if err := db.First(&target, req.Id).Error; err != nil {
		return nil, err
	}

	var metadata map[string]string
	if target.Metadata != "" {
		if err := json.Unmarshal([]byte(target.Metadata), &metadata); err != nil {
			metadata = make(map[string]string)
		}
	}

	return &pb.Target{
		Id:       target.ID,
		Name:     target.Name,
		Type:     target.Type,
		Address:  target.Address,
		Port:     target.Port,
		Interval: target.Interval,
		Metadata: metadata,
		Enabled:  target.Enabled,
	}, nil
}

func (s *Server) ListMonitors(ctx context.Context, req *pb.Empty) (*pb.TargetList, error) {
	db := database.GetDB()

	var targets []models.MonitorTarget
	if err := db.Find(&targets).Error; err != nil {
		return nil, err
	}

	var pbTargets []*pb.Target
	for _, target := range targets {
		var metadata map[string]string
		if target.Metadata != "" {
			if err := json.Unmarshal([]byte(target.Metadata), &metadata); err != nil {
				metadata = make(map[string]string)
			}
		}

		pbTargets = append(pbTargets, &pb.Target{
			Id:       target.ID,
			Name:     target.Name,
			Type:     target.Type,
			Address:  target.Address,
			Port:     target.Port,
			Interval: target.Interval,
			Metadata: metadata,
			Enabled:  target.Enabled,
		})
	}

	return &pb.TargetList{
		Targets: pbTargets,
	}, nil
}

func (s *Server) GetMonitorStatus(ctx context.Context, req *pb.MonitorID) (*pb.MonitorStatus, error) {
	status, err := s.monitorService.GetStatus(req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.MonitorStatus{
		Id:               status.ID,
		Status:           status.Status,
		ResponseTime:     status.ResponseTime,
		Message:          status.Message,
		CheckedAt:        status.CheckedAt.Unix(),
		UptimePercentage: int32(status.UptimePercentage),
	}, nil
}

func (s *Server) ListMonitorStatus(ctx context.Context, req *pb.Empty) (*pb.MonitorStatusList, error) {
	statuses := s.monitorService.ListStatus()

	var pbStatuses []*pb.MonitorStatus
	for _, status := range statuses {
		pbStatuses = append(pbStatuses, &pb.MonitorStatus{
			Id:               status.ID,
			Status:           status.Status,
			ResponseTime:     status.ResponseTime,
			Message:          status.Message,
			CheckedAt:        status.CheckedAt.Unix(),
			UptimePercentage: int32(status.UptimePercentage),
		})
	}

	return &pb.MonitorStatusList{
		Statuses: pbStatuses,
	}, nil
}

func (s *Server) QueryIPGeo(ctx context.Context, req *pb.IPRequest) (*pb.IPGeoResponse, error) {
	ipgeoService := NewIPGeoService()

	result, err := ipgeoService.QueryIP(req.Ip)
	if err != nil {
		return nil, err
	}

	return &pb.IPGeoResponse{
		Ip:        result.IP,
		Country:   result.Country,
		Region:    result.Region,
		City:      result.City,
		Isp:       result.ISP,
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
	}, nil
}

func StartServer(addr string, monitorService *monitor.Service) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	server := NewServer(monitorService)

	pb.RegisterMonitorServiceServer(s, server)
	pb.RegisterIPGeoServiceServer(s, server)

	log.Printf("gRPC server listening on %s", addr)

	return s.Serve(lis)
}