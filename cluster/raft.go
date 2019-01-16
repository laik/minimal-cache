package cluster

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft-boltdb"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpclog "google.golang.org/grpc/grpclog"

	// Static files
	"github.com/laik/minimal-cache/insecure"
	_ "github.com/laik/minimal-cache/statik"

	//
	"github.com/laik/minimal-cache/api"
	command "github.com/laik/minimal-cache/command"
	"github.com/laik/minimal-cache/config"
	"github.com/laik/minimal-cache/raw"
)

const (
	raftApplyTimeout     = 500 * time.Millisecond
	raftLogCacheSize     = 512
	raftMaxPoolSize      = 3
	raftTransportTimeout = 10 * time.Second
	raftSnapshotsRetain  = 3
	raftDBFile           = "raft.db"
	leaderIPMetaKey      = "_leaderIP"
)

var _ = api.MinimalCacheServer(&RaftNode{})

//metaStore is used for storing meta information.
type metaStore interface {
	//SetMeta puts a new value at the given key.
	SetMeta(string, string) error
	//GetMeta get a value at the given key.
	GetMeta(string) (string, error)
	//KeysMeta returns all stored metadata key.
	KeysMeta() ([]string, error)
	//AllMeta return all atored metadata
	AllMeta() (map[string]string, error)
	//RestoreMeta replaces current metadata with the given one.
	RestoreMeta(map[string]string) error
}

//dataStore is used to access stored data.
type dataStore interface {
	Set(string, string) error
	// Get get a value at the given key.
	Get(string) (string, error)
	//Del deletes the given key.
	Del(string) error
	//Keys list all item name
	Keys() ([]string, error)
	//All return all stored data
	All() (map[string]string, error)
	//Restore restores current data with the given one.
	Restore(map[string]string) error
}

type RaftNode struct {
	meta metaStore
	data dataStore

	parser command.CommandParser
	opts   *config.Options

	srv    *grpc.Server
	leader *grpc.ClientConn

	log grpclog.LoggerV2

	raft          *raft.Raft //consensus protocol
	raftTransport raft.Transport
}

func NewRaftNode(meta metaStore, data dataStore, parser command.CommandParser, opts *config.Options) *RaftNode {
	rn := &RaftNode{
		meta:   meta,
		data:   data,
		parser: parser,
		opts:   opts,
	}
	rn.log = grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, os.Stderr)
	grpclog.SetLoggerV2(rn.log)
	return rn
}

// RaftNode creates a new cluster and runs the server.
func (this *RaftNode) BootstrapCluster() error {
	if err := this.setupRaft(); err != nil {
		return fmt.Errorf("could not to setup raft node: %v", err)
	}
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(this.opts.Raft),
				Address: this.raftTransport.LocalAddr(),
			},
		},
	}
	this.raft.BootstrapCluster(configuration)
	return this.start(this.opts.Tcp, this.opts.Http)
}

//Stop stops a grpc server.
func (this *RaftNode) Stop() error {
	this.srv.Stop()
	if this.leader != nil {
		return this.leader.Close()
	}
	return nil
}

func (this *RaftNode) setupRaft() (err error) {
	absDir, err := filepath.Abs(this.opts.DataDir)
	if err != nil {
		return err
	}
	this.opts.DataDir = absDir

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(this.opts.Raft)
	config.Logger = log.New(os.Stderr, "RAFT: ", log.Ldate|log.Ltime)

	addr, err := net.ResolveTCPAddr("tcp", this.opts.Raft)
	if err != nil {
		return fmt.Errorf("could not resolve tcp address: %v", err)
	}

	this.raftTransport, err = raft.NewTCPTransport(this.opts.Raft, addr, raftMaxPoolSize, raftTransportTimeout, os.Stderr)
	if err != nil {
		return fmt.Errorf("could create create TCP transport: %v", err)
	}

	snapshots, err := raft.NewFileSnapshotStore(this.opts.DataDir, raftSnapshotsRetain, os.Stderr)
	if err != nil {
		return fmt.Errorf("could not create file snapshot store: %v", err)
	}

	store, err := raftboltdb.NewBoltStore(filepath.Join(this.opts.DataDir, raftDBFile))
	if err != nil {
		return fmt.Errorf("could not create bolt store: %v", err)
	}

	logStore, err := raft.NewLogCache(raftLogCacheSize, store)
	if err != nil {
		return fmt.Errorf("could not create log cache: %v", err)
	}

	this.raft, err = raft.NewRaft(config, newFsm(this, this.log), logStore, store, snapshots, this.raftTransport)
	if err != nil {
		return fmt.Errorf("could not create raft node: %v", err)
	}

	go this.whenLeaderChanged(
		this.updateLeaderIP,    // Change update metadata leader ip
		this.controlLeaderConn, // Obtain gRPC client to leader connection
	)
	return nil
}

//whenLeaderChanged executes given functions when a leader in the cluster changed.
func (this *RaftNode) whenLeaderChanged(funcs ...func(isLeader bool) error) {
	for isLeader := range this.raft.LeaderCh() {
		wg := new(sync.WaitGroup)
		wg.Add(len(funcs))
		for _, f := range funcs {
			go func(f func(isLeader bool) error) {
				defer wg.Done()
				if err := f(isLeader); err != nil {
					this.log.Infof("[WARN] server: error while executing function when leader changed: %v", err)
				}
			}(f)
		}
		wg.Wait()
	}
}

func (this *RaftNode) isLeader() bool {
	return this.raft.State() == raft.Leader
}

func (this *RaftNode) leaderConn() (*grpc.ClientConn, error) {
	if this.leader != nil {
		return this.leader, nil
	}
	leaderIP, err := this.meta.GetMeta(leaderIPMetaKey)
	if err != nil {
		return nil, fmt.Errorf("could not get leader ip from meta store: %v", err)
	}
	conn, err := gRPCConn(leaderIP)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func (this *RaftNode) controlLeaderConn(isLeader bool) error {
	var err error
	if isLeader {
		if this.leader != nil {
			err = this.leader.Close()
			this.leader = nil
		}
		return err
	}
	this.leader, err = this.leaderConn()
	if err != nil {
		return fmt.Errorf("could not connect to leader: %v", err)
	}
	return nil

}

func (this *RaftNode) handleSetMetaValueRequest(req *api.UpdateMetadataRequest) (*api.UpdateMetadataResponse, error) {
	if err := this.meta.SetMeta(req.Key, req.Value); err != nil {
		return nil, fmt.Errorf("could not save meta %q: %v", req.Key, err)
	}
	return &api.UpdateMetadataResponse{}, nil
}

func (this *RaftNode) handleExecuteCommandRequest(req *api.ExecuteCommandRequest) (*api.ExecuteCommandResponse, error) {
	cmd, args, err := this.parser.Parse(req.Command)
	if err != nil {
		return nil, err
	}
	res := cmd.Execute(args...)
	resp, err := this.createResponse(res)
	if err != nil {
		return nil, fmt.Errorf("could not create response: %v", err)
	}
	return resp, nil
}

func (this *RaftNode) createResponse(res command.Reply) (*api.ExecuteCommandResponse, error) {
	apiRes := new(api.ExecuteCommandResponse)

	switch t := res.(type) {
	case *command.NilReply:
		apiRes.Reply = api.NilCommandReply
	case *command.OkReply:
		apiRes.Reply = api.OkCommandReply
	case *command.StringReply:
		apiRes.Reply = api.StringCommandReply
		apiRes.Item = t.Message
	case *command.SliceReply:
		apiRes.Reply = api.SliceCommandReply
		apiRes.Items = t.Message
	case *command.ErrReply:
		apiRes.Reply = api.ErrCommandReply
		apiRes.Item = fmt.Sprintf("%v", t.Message)
	default:
		return nil, fmt.Errorf("unsupported type %T", res)
	}
	return apiRes, nil
}

func newApplyMetadataFSMCommand(key, value string) (*api.FSMCommand, error) {
	return newFSMCommand(api.FSMApplyMetadata, &api.UpdateMetadataRequest{Key: key, Value: value})
}

func newExecuteFSMCommand(command string) (*api.FSMCommand, error) {
	return newFSMCommand(api.FSMApplyCommand, &api.ExecuteCommandRequest{Command: command})
}

func newFSMCommand(t api.FSMCommandType, req proto.Message) (*api.FSMCommand, error) {
	cmd := &api.FSMCommand{Type: t}
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not marhal proto request: %v", err)
	}
	cmd.Command = raw.Raw(b)
	return cmd, nil
}

func (this *RaftNode) updateLeaderIP(isReader bool) error {
	if !isReader {
		return nil
	}

	cmd, err := newApplyMetadataFSMCommand(leaderIPMetaKey, this.opts.Tcp)
	if err != nil {
		return fmt.Errorf("could not create set meta fsm command: %v", err)
	}

	b, err := proto.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("could not marshal fsm command: %v", err)
	}

	future := this.raft.Apply(b, raftApplyTimeout)

	if err = future.Error(); err != nil {
		return fmt.Errorf("could not apply set meta value request: %v", err)
	}

	if err, ok := future.Response().(error); ok {
		return fmt.Errorf("could not apply set meta value request: %v", err)
	}
	return nil

}

func gRPCConn(addr string) (*grpc.ClientConn, error) {
	dialAddr := fmt.Sprintf("passthrough://%s/%s", addr, addr)
	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("could not dial %s: %v", addr, err)
	}
	return conn, err
}
