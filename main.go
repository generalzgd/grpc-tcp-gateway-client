/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: main.go
 * @time: 2019/9/6 13:33
 */
package main

import (
	`encoding/binary`
	`errors`
	`flag`
	`fmt`
	`io`
	`net`
	`sync`
	`sync/atomic`
	`time`

	`github.com/astaxie/beego/logs`
	gwproto `github.com/generalzgd/grpc-tcp-gateway-proto/goproto`
	`github.com/generalzgd/grpc-tcp-gateway/codec`
	`github.com/golang/protobuf/proto`

	`github.com/generalzgd/grpc-tcp-gateway-client/sub`
)

var (
	seqSeed = uint32(0)

	clientNum = flag.Int("num", 1, "client num input for test")
	gwAddr = flag.String("addr", "test.zhanqi.tv:8989", "测试网关地址")
)

func init() {
	logger := logs.GetBeeLogger()
	logger.SetLevel(logs.LevelInfo)
	logger.SetLogger(logs.AdapterConsole)
	logger.SetLogger(logs.AdapterFile, `{"filename":"logs/file.log","level":7,"maxlines":1024000000,"maxsize":1024000000,"daily":true,"maxdays":7}`)
	logger.EnableFuncCallDepth(true)
	logger.SetLogFuncCallDepth(3)
	logger.Async(100000)
}

func main() {
	flag.Parse()
	//
	wg := sync.WaitGroup{}
	for i:=0; i< *clientNum; i++ {
		wg.Add(1)
		go func() {
			quit := make(chan struct{})
			defer func() {
				close(quit)
				wg.Done()
				recover()
			}()

			conn, err := net.Dial("tcp", *gwAddr)
			if err != nil {
				logs.Error("tcp dial fail. %v", err)
				return
			}
			defer conn.Close()

			go readloop(conn, quit)

			go heartbeat(conn, quit)

			for {
				select {
				case <-quit:
					return
				default:
				}
				req := &gwproto.Method1Request{}
				sendPacket(req, conn, gwproto.GetIdByMsgObj(req))
				time.Sleep(time.Second)

				req2 := &gwproto.Method1Request{}
				sendPacket(req2, conn, gwproto.GetIdByMsgObj(req2))
				time.Sleep(time.Second)
			}
		}()
	}

	wg.Wait()
	logs.Info("tcp gateway client test quit")
	logs.GetBeeLogger().Flush()
}

func heartbeat(conn net.Conn, quit chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		close(quit)
		recover()
	}()
	for {
		select {
		case <-ticker.C:
			beat := &gwproto.HeartBeat{}
			sendPacket(beat, conn, 1)
		case <-quit:
			return
		}
	}
}

func readloop(conn net.Conn, quit chan struct{}) {
	headData := [codec.PACK_HEAD_SIZE]byte{}
	defer func() {
		close(quit)
		recover()
	}()
	for {
		select {
		case <-quit:
			return
		default:
		}

		head := headData[:]
		memSet(head, 0)

		if _, err := io.ReadFull(conn, head); err != nil {
			logs.Error("define.ReadFullErr")
			return
		}
		length := int(binary.LittleEndian.Uint16(head[0:]))
		if length > 32*1024 {
			logs.Error("define.TooLargeErr")
			return
		}
		buff := make([]byte, codec.PACK_HEAD_SIZE+length)
		copy(buff, head)
		if _, err := io.ReadFull(conn, buff[codec.PACK_HEAD_SIZE:]); err != nil {
			logs.Error("define.ReadFullErr")
			return
		}
		//
		pack := codec.GateClientPack{}
		pack.Unserialize(buff)
		logs.Info("receive pack:", fmt.Sprintf("head:%v,Body:%v", pack.GateClientPackHead, sub.FormatObj(pack.Id, pack.Body)))
	}
}

func memSet(mem []byte, v byte) {
	l := len(mem)
	for i := 0; i < l; i++ {
		mem[i] = v
	}
}

func sendPacket(msg proto.Message, conn net.Conn, id uint16) error {
	if conn == nil {
		return errors.New("tcp connection empty")
	}
	bts, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	pack := codec.GateClientPack{
		GateClientPackHead: codec.GateClientPackHead{
			Length: uint16(len(bts)),
			Seq:    uint16(atomic.AddUint32(&seqSeed, 1)),
			Codec:  0,
			Id:     id,
		},
		Body: bts,
	}
	_, err = conn.Write(pack.Serialize())

	logs.Info("send pack: ", fmt.Sprintf("head:%v,Body:%v", pack.GateClientPackHead, sub.FormatObj(pack.Id, pack.Body)))
	return err
}
