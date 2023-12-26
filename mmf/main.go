package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/utah-KT/open-match-tutorials/config"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	openSlotsKey = "openSlots"
)

// MatchFunctionService implements pb.MatchFunctionServer because
// the RegisterMatchFunctionServer requires.
// https://github.com/googleforgames/open-match/blob/v1.8.0/pkg/pb/matchfunction_grpc.pb.go#L116
type MatchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

func (s *MatchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	profile := req.GetProfile()
	pools := profile.GetPools()
	tickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, pools)
	if err != nil {
		log.Printf("Failed to query tickets, got %s", err.Error())
		return err
	}

	backfills, err := matchfunction.QueryBackfillPools(stream.Context(), s.queryServiceClient, pools)
	if err != nil {
		log.Printf("Failed to query backfills, got %s", err.Error())
		return err
	}

	proposals, err := makeMatches(profile, tickets, backfills)
	if err != nil {
		log.Printf("Failed to makeMatches, got %s", err.Error())
		return err
	}

	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

// TODO: check all pool and add match score for evaluator.
func makeMatches(profile *pb.MatchProfile, tickets map[string][]*pb.Ticket, backfills map[string][]*pb.Backfill) ([]*pb.Match, error) {
	var matches []*pb.Match
	validTickets := tickets[config.Global.Matching.Tag]
	validBackfills := backfills[config.Global.Matching.Tag]
	id := 0
	for _, backfill := range validBackfills {
		if len(validTickets) == 0 {
			break
		}
		openSlots, err := getOpenSlots(backfill)
		if err != nil {
			return nil, err
		}
		matchedCount := openSlots
		ticketsNum := len(validTickets)
		if openSlots > ticketsNum {
			matchedCount = ticketsNum
		}

		matched := validTickets[:matchedCount]
		validTickets = validTickets[matchedCount:]
		err = setOpenSlots(backfill, openSlots-len(matched))
		if err != nil {
			return nil, err
		}
		matches = append(matches, newMatch(id, profile.Name, matched, backfill, false))
		id++
	}

	for {
		ticketsNum := len(validTickets)
		if ticketsNum == 0 {
			break
		}
		matchedCount := config.Global.GameServer.MemberNum
		if config.Global.GameServer.MemberNum > ticketsNum {
			matchedCount = ticketsNum
		}
		matched := validTickets[:matchedCount]
		backfill, err := newBackfill(config.Global.GameServer.MemberNum - matchedCount)
		if err != nil {
			return nil, err
		}
		matches = append(matches, newMatch(id, profile.Name, matched, backfill, true))
		validTickets = validTickets[matchedCount:]
		id++
	}
	return matches, nil
}

func newMatch(id int, profile string, tickets []*pb.Ticket, b *pb.Backfill, allocateDGS bool) *pb.Match {
	return &pb.Match{
		MatchId:            fmt.Sprintf("%s-%s-%d", profile, time.Now().Format("2006-01-02T15:04:05.00"), id),
		MatchProfile:       profile,
		MatchFunction:      config.Global.Mmf.Name,
		Tickets:            tickets,
		Backfill:           b,
		AllocateGameserver: allocateDGS,
	}
}

func newBackfill(openSlots int) (*pb.Backfill, error) {
	if openSlots == 0 {
		return nil, nil
	}
	searchFields := &pb.SearchFields{
		Tags: []string{config.Global.Matching.Tag},
	}
	b := &pb.Backfill{
		SearchFields: searchFields,
		Generation:   0,
		CreateTime:   timestamppb.Now(),
	}

	err := setOpenSlots(b, openSlots)
	return b, err
}

func setOpenSlots(b *pb.Backfill, cnt int) error {
	if b.Extensions == nil {
		b.Extensions = make(map[string]*anypb.Any)
	}
	any, err := anypb.New(&wrapperspb.Int32Value{Value: int32(cnt)})
	if err != nil {
		return fmt.Errorf("failed to wrap %d, got %s", cnt, err.Error())
	}

	b.Extensions[openSlotsKey] = any
	return nil
}

func getOpenSlots(b *pb.Backfill) (int, error) {
	if b.Extensions == nil {
		return 0, fmt.Errorf("backfill(id:%s) doesn't have extensions", b.Id)
	}

	any, ok := b.Extensions[openSlotsKey]
	if !ok {
		return 0, fmt.Errorf("backfill(id:%s) doesn't have open slot field", b.Id)
	}

	var val wrapperspb.Int32Value
	err := any.UnmarshalTo(&val)
	if err != nil {
		return 0, err
	}

	return int(val.Value), nil
}

func main() {
	config.Load()
	conn, err := grpc.Dial(config.Global.OpenMatch.QueryEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Query, got %s", err.Error())
	}
	defer conn.Close()

	mmfService := MatchFunctionService{
		queryServiceClient: pb.NewQueryServiceClient(conn),
	}

	server := grpc.NewServer()
	pb.RegisterMatchFunctionServer(server, &mmfService)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Global.Mmf.Port))
	if err != nil {
		log.Fatalf("TCP net listener initialization failed for port %v, got %s", config.Global.Mmf.Port, err.Error())
	}

	log.Printf("TCP net listener initialized for port %v", config.Global.Mmf.Port)
	err = server.Serve(ln)
	if err != nil {
		log.Fatalf("gRPC serve failed, got %s", err.Error())
	}
}
