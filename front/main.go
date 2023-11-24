package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/utah-KT/open-match-tutorials/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	ompb "open-match.dev/open-match/pkg/pb"
)

const (
	Port               = 54321
	omFrontendEndpoint = "open-match-frontend.open-match.svc.cluster.local:50504"
	defaultPoolTag     = "default"
)

type GameFront struct {
	pb.GameFrontServiceServer
	FrontendClient ompb.FrontendServiceClient
}

func NewGameFrontEnd(conn *grpc.ClientConn) *GameFront {
	return &GameFront{
		FrontendClient: ompb.NewFrontendServiceClient(conn),
	}
}

func (gf *GameFront) requestCreateTicket(name string) (*ompb.Ticket, error) {
	log.Printf("request create ticket (tag: %s)", defaultPoolTag)
	ext, err := createTicketExtensions(name)
	if err != nil {
		return nil, err
	}
	t := &ompb.Ticket{
		SearchFields: &ompb.SearchFields{
			Tags: []string{defaultPoolTag},
		},
		Extensions: ext,
	}
	ticketReq := &ompb.CreateTicketRequest{Ticket: t}
	return gf.FrontendClient.CreateTicket(context.Background(), ticketReq)
}

func createTicketExtensions(name string) (map[string]*anypb.Any, error) {
	e := make(map[string]*anypb.Any)
	any, err := anypb.New(&wrapperspb.StringValue{Value: name})
	if err != nil {
		return nil, fmt.Errorf("failed to wrap %s, got %s", name, err.Error())
	}

	e["name"] = any
	return e, nil
}

func (gf *GameFront) watchAssignments(ticketID string, stream pb.GameFrontService_EntryGameServer) error {
	var assignment *ompb.Assignment
	watchReq := &ompb.WatchAssignmentsRequest{TicketId: ticketID}
	assignmentsStream, err := gf.FrontendClient.WatchAssignments(context.Background(), watchReq)
	defer assignmentsStream.CloseSend()
	if err != nil {
		log.Printf("failed to watch ticket %s, got %s", ticketID, err.Error())
		return err
	}
	for assignment.GetConnection() == "" {
		assignmentsRes, err := assignmentsStream.Recv()
		if err != nil {
			log.Printf("failed to receive assignments, got %s", err.Error())
			return err
		}
		assignment = assignmentsRes.Assignment
	}
	stream.Send(&pb.EntryGameResponse{
		TicketId: ticketID,
	})
	_, err = gf.FrontendClient.DeleteTicket(context.Background(), &ompb.DeleteTicketRequest{TicketId: ticketID})
	return err
}

func (gf *GameFront) EntryGame(req *pb.EntryGameRequest, stream pb.GameFrontService_EntryGameServer) error {
	ticket, err := gf.requestCreateTicket(req.Name)
	if err != nil {
		log.Printf("failed to create ticket, got %s", err.Error())
		return err
	}
	log.Printf("ticket(id=%s) is created.", ticket.Id)
	return gf.watchAssignments(ticket.Id, stream)
}

func main() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	conn, err := grpc.Dial(omFrontendEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Backend, got %s", err.Error())
	}

	defer conn.Close()
	defer ln.Close()
	s := grpc.NewServer()
	gf := NewGameFrontEnd(conn)
	pb.RegisterGameFrontServiceServer(s, gf)
	reflection.Register(s)
	if err = s.Serve(ln); err != nil {
		log.Fatalf("Failed to Serve: %v", err)
	}
}
