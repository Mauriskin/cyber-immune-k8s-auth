package main

import (
	"context"
	"log"
	"net"

	pb "security-controller/security" // путь к protobuf

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedSecurityControllerServer
}

func (s *server) CheckInteraction(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	allowed, reason := IsAllowed(req.SourceDomain, req.TargetDomain, req.Action)
	return &pb.CheckResponse{
		Allowed: allowed,
		Reason:  reason,
	}, nil
}

func main() {
	if err := LoadPolicies(); err != nil {
		log.Fatal("Failed to load policies: ", err)
	}

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatal("Failed to listen: ", err)
	}

	s := grpc.NewServer()
	pb.RegisterSecurityControllerServer(s, &server{})
	log.Println("Security Controller listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to serve: ", err)
	}
}
