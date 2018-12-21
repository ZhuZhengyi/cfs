package repl

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/tiglabs/containerfs/proto"
	"github.com/tiglabs/containerfs/storage"
	"github.com/tiglabs/containerfs/util"
	"github.com/tiglabs/containerfs/util/exporter"
)

var (
	ErrBadNodes       = errors.New("BadNodesErr")
	ErrArgLenMismatch = errors.New("ArgLenMismatchErr")
)

type Packet struct {
	proto.Packet
	followerConns  []*net.TCPConn
	followersAddrs []string
	IsRelase       int32
	Object         interface{}
	TpObject       *exporter.TpMetric
	NeedReply      bool
}

func (p *Packet) AfterTp() (ok bool) {
	p.TpObject.CalcTp()

	return
}

func (p *Packet) BeforeTp(clusterID string) (ok bool) {
	key := fmt.Sprintf("%s_datanode_stream%v", clusterID, p.GetOpMsg())
	p.TpObject = exporter.RegistTp(key)
	return
}

func (p *Packet) resolveFollowersAddr() (err error) {
	defer func() {
		if err != nil {
			p.PackErrorBody(ActionPreparePkg, err.Error())
		}
	}()
	if len(p.Arg) < int(p.Arglen) {
		err = ErrArgLenMismatch
		return
	}
	str := string(p.Arg[:int(p.Arglen)])
	followerAddrs := strings.SplitN(str, proto.AddrSplit, -1)
	followerNum := uint8(len(followerAddrs) - 1)
	p.followersAddrs = make([]string, followerNum)
	if followerNum > 0 {
		p.followersAddrs = followerAddrs[:int(followerNum)]
	}
	p.followerConns = make([]*net.TCPConn, followerNum)
	if p.RemainFollowers < 0 {
		err = ErrBadNodes
		return
	}

	return
}

func (p *Packet) forceDestoryFollowerConnects() {
	for i := 0; i < len(p.followerConns); i++ {
		gConnPool.ForceDestory(p.followerConns[i], p.followersAddrs[i])
	}
}

func (p *Packet) PutConnectsToPool() {
	for i := 0; i < len(p.followerConns); i++ {
		gConnPool.PutConnect(p.followerConns[i], NoCloseConnect)
	}
}

func NewPacket() (p *Packet) {
	p = new(Packet)
	p.Magic = proto.ProtoMagic
	p.StartT = time.Now().UnixNano()
	p.NeedReply = true
	return
}

func (p *Packet) IsMasterCommand() bool {
	switch p.Opcode {
	case
		proto.OpDataNodeHeartbeat,
		proto.OpLoadDataPartition,
		proto.OpCreateDataPartition,
		proto.OpDeleteDataPartition,
		proto.OpOfflineDataPartition:
		return true
	}
	return false
}

func (p *Packet) isForwardPacket() bool {
	r := p.RemainFollowers > 0
	return r
}

func NewGetAllWaterMarker(partitionID uint64, extentType uint8) (p *Packet) {
	p = new(Packet)
	p.Opcode = proto.OpGetAllWaterMark
	p.PartitionID = partitionID
	p.Magic = proto.ProtoMagic
	p.ReqID = proto.GeneratorRequestID()
	p.ExtentMode = extentType

	return
}

func NewExtentRepairReadPacket(partitionID uint64, extentID uint64, offset, size int) (p *Packet) {
	p = new(Packet)
	p.ExtentID = extentID
	p.PartitionID = partitionID
	p.Magic = proto.ProtoMagic
	p.ExtentOffset = int64(offset)
	p.Size = uint32(size)
	p.Opcode = proto.OpExtentRepairRead
	p.ExtentMode = proto.NormalExtentMode
	p.ReqID = proto.GeneratorRequestID()

	return
}

func NewStreamReadResponsePacket(requestID int64, partitionID uint64, extentID uint64) (p *Packet) {
	p = new(Packet)
	p.ExtentID = extentID
	p.PartitionID = partitionID
	p.Magic = proto.ProtoMagic
	p.Opcode = proto.OpOk
	p.ReqID = requestID
	p.ExtentMode = proto.NormalExtentMode

	return
}

func NewNotifyExtentRepair(partitionID uint64) (p *Packet) {
	p = new(Packet)
	p.Opcode = proto.OpNotifyExtentRepair
	p.PartitionID = partitionID
	p.Magic = proto.ProtoMagic
	p.ExtentMode = proto.NormalExtentMode
	p.ReqID = proto.GeneratorRequestID()

	return
}

func (p *Packet) IsErrPacket() bool {
	return p.ResultCode != proto.OpOk
}

func (p *Packet) getErrMessage() (m string) {
	return fmt.Sprintf("req(%v) err(%v)", p.GetUniqueLogId(), string(p.Data[:p.Size]))
}

var (
	ErrorUnknownOp = errors.New("unknown opcode")
)

func (p *Packet) identificationErrorResultCode(errLog string, errMsg string) {
	if strings.Contains(errLog, ActionReceiveFromFollower) || strings.Contains(errLog, ActionSendToFollowers) ||
		strings.Contains(errLog, ConnIsNullErr) || strings.Contains(errLog, ActionCheckAndAddInfos) {
		p.ResultCode = proto.OpIntraGroupNetErr
		return
	}

	if strings.Contains(errMsg, storage.ErrorParamMismatch.Error()) ||
		strings.Contains(errMsg, ErrorUnknownOp.Error()) {
		p.ResultCode = proto.OpArgMismatchErr
	} else if strings.Contains(errMsg, storage.ErrorExtentNotFound.Error()) ||
		strings.Contains(errMsg, storage.ErrorExtentHasDelete.Error()) {
		p.ResultCode = proto.OpNotExistErr
	} else if strings.Contains(errMsg, storage.ErrSyscallNoSpace.Error()) {
		p.ResultCode = proto.OpDiskNoSpaceErr
	} else if strings.Contains(errMsg, storage.ErrorAgain.Error()) {
		p.ResultCode = proto.OpAgain
	} else if strings.Contains(errMsg, storage.ErrNotLeader.Error()) {
		p.ResultCode = proto.OpNotLeaderErr
	} else if strings.Contains(errMsg, storage.ErrorExtentNotFound.Error()) {
		if p.Opcode != proto.OpWrite {
			p.ResultCode = proto.OpNotExistErr
		} else {
			p.ResultCode = proto.OpIntraGroupNetErr
		}
	} else {
		p.ResultCode = proto.OpIntraGroupNetErr
	}
}

func (p *Packet) PackErrorBody(action, msg string) {
	p.identificationErrorResultCode(action, msg)
	if p.ResultCode == proto.OpDiskNoSpaceErr || p.ResultCode == proto.OpDiskErr {
		p.ResultCode = proto.OpIntraGroupNetErr
	}
	p.Size = uint32(len([]byte(action + "_" + msg)))
	p.Data = make([]byte, p.Size)
	copy(p.Data[:int(p.Size)], []byte(action+"_"+msg))
}
func (p *Packet) ReadFull(c net.Conn, readSize int) (err error) {
	if p.Opcode == proto.OpWrite && readSize == util.BlockSize {
		p.Data, _ = proto.Buffers.Get(util.BlockSize)
	} else {
		p.Data = make([]byte, readSize)
	}
	_, err = io.ReadFull(c, p.Data[:readSize])
	return
}

func (p *Packet) isReadOperation() bool {
	return p.Opcode == proto.OpStreamRead || p.Opcode == proto.OpRead || p.Opcode == proto.OpExtentRepairRead
}

func (p *Packet) ReadFromConnFromCli(c net.Conn, deadlineTime time.Duration) (err error) {
	if deadlineTime != proto.NoReadDeadlineTime {
		c.SetReadDeadline(time.Now().Add(deadlineTime * time.Second))
	}
	header, err := proto.Buffers.Get(util.PacketHeaderSize)
	if err != nil {
		header = make([]byte, util.PacketHeaderSize)
	}
	defer proto.Buffers.Put(header)
	if _, err = io.ReadFull(c, header); err != nil {
		return
	}
	if err = p.UnmarshalHeader(header); err != nil {
		return
	}

	if p.Arglen > 0 {
		if err = proto.ReadFull(c, &p.Arg, int(p.Arglen)); err != nil {
			return
		}
	}

	if p.Size < 0 {
		return
	}
	size := p.Size
	if p.isReadOperation() && p.ResultCode == proto.OpInitResultCode {
		size = 0
	}
	return p.ReadFull(c, int(size))
}
