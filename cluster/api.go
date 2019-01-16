package cluster

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/laik/minimal-cache/api"
	command "github.com/laik/minimal-cache/command"
	context "golang.org/x/net/context"
)

func (this *RaftNode) ExecuteCommand(ctx context.Context, req *api.ExecuteCommandRequest) (resp *api.ExecuteCommandResponse, err error) {
	cmd, args, err := this.parser.Parse(req.Command)
	if err != nil {
		if err == command.ErrCommandNotFound {
			return &api.ExecuteCommandResponse{
				Reply: api.ErrCommandReply,
				Item:  fmt.Sprintf("command %q not found", req.Command),
			}, nil
		}
		return nil, fmt.Errorf("could not parse command: %v", err)
	}

	resp = &api.ExecuteCommandResponse{}

	switch cmd.(type) {
	case *command.Set, *command.Del:
		// If the command is modify data, need have to leader node update
		if !this.isLeader() {
			conn, err := this.leaderConn()
			if err != nil {
				return nil, err
			}
			return api.NewMinimalCacheClient(conn).ExecuteCommand(ctx, req)
		}

		fsmCmd, err := newExecuteFSMCommand(req.Command)
		if err != nil {
			return nil, fmt.Errorf("could not create execute fsm command: %v", err)
		}

		b, err := proto.Marshal(fsmCmd)
		if err != nil {
			return nil, fmt.Errorf("could not marshal fsm command: %v", err)
		}

		future := this.raft.Apply(b, raftApplyTimeout)
		if err = future.Error(); err != nil {
			return nil, fmt.Errorf("could not apply raft log entry: %v", err)
		}

		if err, ok := future.Response().(error); ok {
			return nil, fmt.Errorf("could not apply raft log entry: %v", err)
		}

		err = proto.Unmarshal(future.Response().([]byte), resp)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal applied response: %v", err)
		}

	default:
		res := cmd.Execute(args...)
		// if err != nil {
		// 	return nil, fmt.Errorf("could not execute command: %v", err)
		// }
		resp, err = this.createResponse(res)
	}
	return
}

func (this *RaftNode) AddToCluster(ctx context.Context, req *api.AddToClusterRequest) (resp *api.AddToClusterResponse, err error) {
	if !this.isLeader() {
		conn, err := this.leaderConn()
		if err != nil {
			return nil, err
		}
		return api.NewMinimalCacheClient(conn).AddToCluster(ctx, req)
	}
	configFuture := this.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return nil, fmt.Errorf("failed to get raft configuration: %v", err)
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(req.Id) || srv.Address == raft.ServerAddress(req.Addr) {
			if srv.Address == raft.ServerAddress(req.Addr) && srv.ID == raft.ServerID(req.Id) {
				return &api.AddToClusterResponse{}, nil
			}

			future := this.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return nil, fmt.Errorf("error removing existing node %s at %s: %s", req.Id, req.Addr, err)
			}
		}
	}

	f := this.raft.AddVoter(raft.ServerID(req.Id), raft.ServerAddress(req.Addr), 0, 0)
	if f.Error() != nil {
		return nil, f.Error()
	}
	return &api.AddToClusterResponse{}, nil
}

func (this *RaftNode) RemoveOnCluster(ctx context.Context, req *api.RemoveClusterRequest) (*api.RemoveClusterResponse, error) {
	if !this.isLeader() {
		conn, err := this.leaderConn()
		if err != nil {
			return nil, err
		}
		return api.NewMinimalCacheClient(conn).RemoveOnCluster(ctx, req)
	}

	configFuture := this.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return nil, fmt.Errorf("failed to get raft configuration: %v", err)
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(req.Id) {
			this.log.Infof("removing existing node %s at %s", req.Id, req.Addr)
			future := this.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return nil, fmt.Errorf("error removing existing node %s at %s: %s", req.Id, req.Addr, err)
			}
		}
	}
	return &api.RemoveClusterResponse{}, nil
}

func (this *RaftNode) MemberList(ctx context.Context, req *api.MemeberRequest) (*api.MemberResponse, error) {
	memberResponse := &api.MemberResponse{
		RaftNodes: make([]*api.RaftMemberInfo, 0),
	}
	configFuture := this.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return nil, fmt.Errorf("failed to get raft configuration: %v", err)
	}
	for _, srv := range configFuture.Configuration().Servers {
		serverState := srv.Suffrage.String()
		state := api.NodeState_value[strings.ToUpper(serverState)]

		memberResponse.RaftNodes = append(
			memberResponse.RaftNodes,
			&api.RaftMemberInfo{
				Id:    string(srv.ID),
				Addr:  string(srv.Address),
				State: api.NodeState(state),
			})
	}
	return memberResponse, nil
}

//JoinCluster joins to an existing cluster and runs the server.
func (this *RaftNode) JoinCluster(joinAddr string) error {
	if err := this.setupRaft(); err != nil {
		return fmt.Errorf("could not to setup raft node: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- this.start(this.opts.Tcp, this.opts.Http)
	}()

	conn, err := gRPCConn(joinAddr)
	if err != nil {
		return err
	}
	client := api.NewMinimalCacheClient(conn)
	req := &api.AddToClusterRequest{
		Id:   this.opts.Raft,
		Addr: this.opts.Raft,
	}
	if _, err = client.AddToCluster(context.Background(), req); err != nil {
		return fmt.Errorf("could not add a new node to the cluster: %v", err)
	}
	conn.Close()

	return <-errCh
}
