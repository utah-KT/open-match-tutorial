package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/utah-KT/open-match-tutorials/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
	ompb "open-match.dev/open-match/pkg/pb"
)

const (
	omBackendEndpoint        = "open-match-backend.open-match.svc.cluster.local:50505"
	functionHostName         = "open-match-tutorial-mmf.open-match-test.svc.cluster.local"
	functionPort       int32 = 50502
	gameServerEndpoint       = "open-match-tutorial-gameserver.open-match-test.svc.cluster.local:7654"
	openSlotsKey             = "openSlots"
	defaultPoolTag           = "default"
)

type Director struct {
	BackendClient ompb.BackendServiceClient
}

func NewDirector(conn *grpc.ClientConn) *Director {
	return &Director{
		BackendClient: ompb.NewBackendServiceClient(conn),
	}
}

func (d *Director) Fetch(p *ompb.MatchProfile) ([]*ompb.Match, error) {
	req := &ompb.FetchMatchesRequest{
		Config: &ompb.FunctionConfig{
			Host: functionHostName,
			Port: functionPort,
			Type: ompb.FunctionConfig_GRPC,
		},
		Profile: p,
	}

	stream, err := d.BackendClient.FetchMatches(context.Background(), req)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var result []*ompb.Match
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		result = append(result, resp.GetMatch())
	}

	return result, nil
}

func (d *Director) Assign(matches []*ompb.Match) error {
	for _, match := range matches {
		backfill := match.GetBackfill()
		if match.AllocateGameserver {
			err := d.Allocate(match.Tickets, backfill)
			if err != nil {
				return err
			}
		}
		// request tickets assignment because AcknowledgeBackfill was never called.
		if backfill == nil {
			ticketIDs := []string{}
			for _, t := range match.GetTickets() {
				ticketIDs = append(ticketIDs, t.Id)
			}
			req := &ompb.AssignTicketsRequest{
				Assignments: []*ompb.AssignmentGroup{
					{
						TicketIds: ticketIDs,
						Assignment: &ompb.Assignment{
							Connection: gameServerEndpoint,
						},
					},
				},
			}
			_, err := d.BackendClient.AssignTickets(context.Background(), req)
			if err != nil {
				return fmt.Errorf("AssignTickets failed for match %s, got %s", match.GetMatchId(), err.Error())
			}
		}
	}
	return nil
}

func (d *Director) Allocate(tickets []*ompb.Ticket, backfill *ompb.Backfill) error {
	backfillID := ""
	if backfill != nil {
		backfillID = backfill.Id
		log.Printf("with %s", backfillID)
	}
	conn, err := grpc.Dial(gameServerEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("Failed to connect to game server got %s", err.Error())
	}
	defer conn.Close()
	cli := pb.NewGameServerServiceClient(conn)
	req := &pb.AllocateRequest{
		Tickets:    d.ticketsToMemberTickets(tickets),
		BackfillId: backfillID,
	}
	_, err = cli.Allocate(context.Background(), req)
	if err != nil {
		return fmt.Errorf("Failed to allocate got %s", err.Error())
	}
	return nil
}

func (d *Director) ticketsToMemberTickets(tickets []*ompb.Ticket) []*pb.MemberTicket {
	members := make([]*pb.MemberTicket, 0)
	for _, t := range tickets {
		name, err := getName(t)
		if err != nil {
			log.Printf("failed to get name :%s", err.Error())
		}
		m := &pb.MemberTicket{
			TicketId: t.Id,
			Name:     name,
		}
		members = append(members, m)
	}
	return members
}

func getName(t *ompb.Ticket) (string, error) {
	if t.Extensions == nil {
		return "", fmt.Errorf("ticket(id:%s) doesn't have extensions", t.Id)
	}

	any, ok := t.Extensions["name"]
	if !ok {
		return "", fmt.Errorf("ticket(id:%s) doesn't have name field", t.Id)
	}

	var val wrapperspb.StringValue
	err := any.UnmarshalTo(&val)
	if err != nil {
		return "", err
	}

	return val.GetValue(), nil
}

func main() {
	conn, err := grpc.Dial(omBackendEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Backend, got %s", err.Error())
	}

	defer conn.Close()
	director := NewDirector(conn)
	tag := &ompb.TagPresentFilter{Tag: defaultPoolTag}
	pool := &ompb.Pool{
		Name:              "default",
		TagPresentFilters: []*ompb.TagPresentFilter{tag},
	}
	profile := &ompb.MatchProfile{
		Name:  "default",
		Pools: []*ompb.Pool{pool},
	}
	for range time.Tick(250 * time.Millisecond) {
		matches, err := director.Fetch(profile)
		if err != nil {
			log.Printf("Failed to fetch matches for profile %v, got %s", profile.GetName(), err.Error())
			continue // wait mmf...
		}
		err = director.Assign(matches)
		if err != nil {
			log.Printf("Failed to assign matches, got %s", err.Error())
		}
	}
}
