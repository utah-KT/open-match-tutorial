package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/utah-KT/open-match-tutorials/config"
	pb "github.com/utah-KT/open-match-tutorials/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	ompb "open-match.dev/open-match/pkg/pb"
)

type Room struct {
	Members map[string]*pb.Member
	Ready   bool
}

type GameServer struct {
	pb.GameServerServiceServer
	RoomState              *Room
	RoomCh                 chan struct{}
	MemberChannels         map[string]chan *pb.JoinResponse
	TimerCh                chan struct{}
	BackfillID             string
	LastBackfillGeneration int64
	FrontendClient         ompb.FrontendServiceClient
	RoomStateMutex         sync.Mutex
	BackfillStateMutex     sync.Mutex
}

func NewGameServer(conn *grpc.ClientConn) *GameServer {
	gs := &GameServer{
		FrontendClient: ompb.NewFrontendServiceClient(conn),
	}
	gs.initRoom()
	return gs
}

func (gs *GameServer) initRoom() {
	room := &Room{
		Members: map[string]*pb.Member{},
		Ready:   false,
	}
	gs.RoomStateMutex.Lock()
	gs.BackfillStateMutex.Lock()
	gs.RoomState = room
	gs.RoomCh = make(chan struct{})
	gs.MemberChannels = map[string]chan *pb.JoinResponse{}
	gs.TimerCh = make(chan struct{})
	gs.BackfillID = ""
	gs.LastBackfillGeneration = 0
	gs.BackfillStateMutex.Unlock()
	gs.RoomStateMutex.Unlock()
	go gs.watchRoomState()
}

func (gs *GameServer) watchRoomState() {
	defer close(gs.RoomCh)
	for {
		select {
		case _ = <-gs.RoomCh:
			res := gs.entryResponseFromRoomState()
			for member, memberCh := range gs.MemberChannels {
				select {
				case memberCh <- res:
					log.Printf("Send state %v to %s", res, member)
				}
			}
			if res.Ready {
				log.Printf("Room is ready")
				gs.TimerCh <- struct{}{} // notify stop
				gs.deleteBackfill()
				gs.initRoom()
				return
			}
		}
	}
}

func (gs *GameServer) deleteBackfill() error {
	if gs.BackfillID == "" {
		return nil
	}
	gs.BackfillStateMutex.Lock()
	defer gs.BackfillStateMutex.Unlock()
	_, err := gs.FrontendClient.DeleteBackfill(context.Background(), &ompb.DeleteBackfillRequest{BackfillId: gs.BackfillID})
	if err != nil {
		return err
	}
	gs.BackfillID = ""
	return nil
}

func (gs *GameServer) ackAndcheckTimeout() {
	ticker := time.NewTicker(500 * time.Millisecond)
	timer := time.After(time.Duration(config.Global.GameServer.Timeout) * time.Second)
	defer close(gs.TimerCh)
	log.Println("start timer")
	for {
		select {
		case <-gs.TimerCh:
			ticker.Stop()
			log.Println("stop by channel")
			return
		case <-ticker.C:
			if gs.BackfillID != "" {
				req := &ompb.AcknowledgeBackfillRequest{
					BackfillId: gs.BackfillID,
					Assignment: &ompb.Assignment{
						Connection: "myEndpoint",
					},
				}
				res, err := gs.FrontendClient.AcknowledgeBackfill(context.Background(), req)
				if err != nil {
					log.Printf("error on AcknowledgeBackfill (%s)", err.Error())
					continue
				}
				gs.RoomStateMutex.Lock()
				gs.BackfillStateMutex.Lock()
				gs.BackfillID = res.Backfill.Id
				for _, ticket := range res.Tickets {
					name, err := getName(ticket)
					if err != nil {
						log.Printf("error on get name (%s)", err.Error())
						continue
					}
					gs.RoomState.Members[ticket.Id] = &pb.Member{Name: name, Ready: false}
				}
				if res.Backfill.Generation > gs.LastBackfillGeneration {
					gs.LastBackfillGeneration = res.Backfill.Generation
					gs.RoomCh <- struct{}{}
				}
				gs.BackfillStateMutex.Unlock()
				gs.RoomStateMutex.Unlock()
			}
		case <-timer:
			ticker.Stop()
			gs.deleteBackfill()
			for i := 1; len(gs.RoomState.Members) < config.Global.GameServer.MemberNum; i++ {
				name := fmt.Sprintf("bot%d", i)
				bot := &pb.Member{
					Name:  name,
					Ready: true,
				}
				gs.addMember(name, bot)
			}
			gs.RoomCh <- struct{}{}
		}
	}
}

func getName(ticket *ompb.Ticket) (string, error) {
	any, ok := ticket.Extensions["name"]
	if !ok {
		return "", fmt.Errorf("ticket %s doesn't have name field", ticket.Id)
	}
	var val wrapperspb.StringValue
	err := any.UnmarshalTo(&val)
	return val.GetValue(), err
}

func (gs *GameServer) addMembers(memberTickets []*pb.MemberTicket) {
	for _, mt := range memberTickets {
		m := &pb.Member{
			Name:  mt.Name,
			Ready: false,
		}
		gs.addMember(mt.TicketId, m)
	}
	gs.RoomCh <- struct{}{}
}

func (gs *GameServer) addMember(ticketID string, member *pb.Member) {
	gs.RoomStateMutex.Lock()
	defer gs.RoomStateMutex.Unlock()
	gs.RoomState.Members[ticketID] = member
	gs.updateRoomReadyWithoutLock()
}

func (gs *GameServer) Allocate(ctx context.Context, req *pb.AllocateRequest) (*emptypb.Empty, error) {
	gs.addMembers(req.Tickets)
	gs.BackfillStateMutex.Lock()
	gs.BackfillID = req.BackfillId
	gs.BackfillStateMutex.Unlock()
	go gs.ackAndcheckTimeout()
	return &emptypb.Empty{}, nil
}

func (gs *GameServer) joinRoom(ticketID string) error {
	gs.RoomStateMutex.Lock()
	defer gs.RoomStateMutex.Unlock()
	if gs.RoomState.Members[ticketID] == nil {
		return fmt.Errorf("ticket id %s is not assigned", ticketID)
	}
	gs.RoomState.Members[ticketID].Ready = true
	gs.updateRoomReadyWithoutLock()
	return nil
}

func (gs *GameServer) updateRoomReadyWithoutLock() {
	roomReady := len(gs.RoomState.Members) == config.Global.GameServer.MemberNum
	if roomReady {
		for _, member := range gs.RoomState.Members {
			roomReady = roomReady && member.Ready
		}
		gs.RoomState.Ready = roomReady
	}
}

func (gs *GameServer) entryResponseFromRoomState() *pb.JoinResponse {
	members := make([]*pb.Member, 0, config.Global.GameServer.MemberNum)
	for _, member := range gs.RoomState.Members {
		members = append(members, member)
	}
	res := &pb.JoinResponse{
		Members: members,
		Ready:   gs.RoomState.Ready,
	}
	return res
}

func (gs *GameServer) Join(req *pb.JoinRequest, stream pb.GameServerService_JoinServer) error {
	err := gs.joinRoom(req.TicketId)
	if err != nil {
		return err
	}
	myCh := make(chan *pb.JoinResponse)
	gs.MemberChannels[req.TicketId] = myCh
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(myCh)
		for {
			select {
			case res := <-myCh:
				stream.Send(res)
				if res.Ready {
					return
				}
			}
		}
	}()
	gs.RoomCh <- struct{}{} // notify
	wg.Wait()
	return nil
}

func main() {
	config.Load()
	ln, err := net.Listen("tcp", ":7654")
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	conn, err := grpc.Dial(config.Global.OpenMatch.FrontendEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to %s, got %s", config.Global.OpenMatch.FrontendEndpoint, err.Error())
	}

	defer conn.Close()
	defer ln.Close()
	s := grpc.NewServer()
	gs := NewGameServer(conn)
	pb.RegisterGameServerServiceServer(s, gs)
	reflection.Register(s)
	if err = s.Serve(ln); err != nil {
		log.Fatalf("Failed to Serve: %v", err)
	}
}
