package cluster

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/laik/minimal-cache/api"
	grpclog "google.golang.org/grpc/grpclog"
)

type fsm struct {
	raftNode *RaftNode
	log      grpclog.LoggerV2
}

func newFsm(r *RaftNode, log grpclog.LoggerV2) *fsm {
	return &fsm{
		raftNode: r,
		log:      log,
	}
}

func (f *fsm) Apply(entry *raft.Log) interface{} {
	var (
		fsmCommand = &api.FSMCommand{}
		err        error
		resp       proto.Message
	)

	if err = proto.Unmarshal(entry.Data, fsmCommand); err != nil {
		return fmt.Errorf("could not unmarshal log entry: %v", err)
	}
	switch fsmCommand.Type {
	case api.FSMApplyMetadata:
		req := &api.UpdateMetadataRequest{}
		if err = proto.Unmarshal([]byte(fsmCommand.Command), req); err != nil {
			return fmt.Errorf("could not unmarshal set meta value request: %v", err)
		}
		if resp, err = f.raftNode.handleSetMetaValueRequest(req); err != nil {
			return err
		}
	case api.FSMApplyCommand:
		req := &api.ExecuteCommandRequest{}
		if err = proto.Unmarshal([]byte(fsmCommand.Command), req); err != nil {
			return fmt.Errorf("could not unmarshal execute command request: %v", err)
		}
		if resp, err = f.raftNode.handleExecuteCommandRequest(req); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized fsm command: %v", fsmCommand.Type)
	}
	b, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("could not marshal response: %v", err)
	}
	return b
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	s := &fsmSnapshot{}
	meta, err := f.raftNode.meta.AllMeta()
	if err != nil {
		return nil, fmt.Errorf("could not get all metadata: %v", err)
	}
	mb, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("could not marshal metadata: %v", err)
	}
	s.meta = mb

	data, err := f.raftNode.data.All()
	if err != nil {
		return nil, fmt.Errorf("could not get all data: %v", err)
	}
	db, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal data: %v", err)
	}
	s.data = db
	return s, nil
}

//Restore implements raft.FSM interface.
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var l int64

	if err := binary.Read(rc, binary.LittleEndian, &l); err != nil {
		return fmt.Errorf("could not read meta length: %v", err)
	}

	mb := make([]byte, l)

	if n, err := io.ReadFull(rc, mb); int64(n) != l || err != nil {
		return fmt.Errorf("could not read meta")
	}

	meta := make(map[string]string)

	if err := json.Unmarshal(mb, &meta); err != nil {
		return fmt.Errorf("could not unmarshal meta: %v", err)
	}

	if err := f.raftNode.meta.RestoreMeta(meta); err != nil {
		return fmt.Errorf("could restore meta: %v", err)
	}

	if err := binary.Read(rc, binary.LittleEndian, &l); err != nil {
		return fmt.Errorf("could not read data length: %v", err)
	}

	db := make([]byte, l)

	if n, err := io.ReadFull(rc, db); int64(n) != l || err != nil {
		return fmt.Errorf("could not read data")
	}

	data := make(map[string]string)
	if err := json.Unmarshal(db, &data); err != nil {
		return fmt.Errorf("could unmarshal data: %v", err)
	}

	if err := f.raftNode.data.Restore(data); err != nil {
		return fmt.Errorf("could not restore data: %v", err)
	}
	return nil
}

type fsmSnapshot struct {
	meta []byte
	data []byte
}

func (fs *fsmSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	defer func() {
		if err == nil {
			err = sink.Close()
		} else {
			err = sink.Cancel()
		}
	}()
	if err := binary.Write(sink, binary.LittleEndian, int64(len(fs.meta))); err != nil {
		return fmt.Errorf("could not write meta length: %v", err)
	}
	if _, err := sink.Write(fs.meta); err != nil {
		return fmt.Errorf("could not write meta: %v", err)
	}
	if err := binary.Write(sink, binary.LittleEndian, int64(len(fs.data))); err != nil {
		return fmt.Errorf("could not write data length: %v", err)
	}
	if _, err := sink.Write(fs.data); err != nil {
		return fmt.Errorf("could not write data: %v", err)
	}
	return nil
}

//Release is invoked when we are finished with the snapshot.
func (fs *fsmSnapshot) Release() {
	//noop
}
