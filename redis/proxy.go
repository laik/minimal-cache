package redis

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/laik/minimal-cache/api"
	"github.com/laik/minimal-cache/insecure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var redis = flag.String("redis", "6379", "redis cli proxy prot")

type RedisCliProxy struct {
	listener net.Listener
	client   api.MinimalCacheClient
}

func NewRedisCliProxy(grpcAddr string) {

	dialAddr := fmt.Sprintf("passthrough://localhost/%s", grpcAddr)

	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial local server with serve redis proxy error: %s\n", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", *redis))

	if err != nil {
		fmt.Fprintf(os.Stdout, "listen redis cli proxy error: %s", err)
	}

	defer listener.Close()

	rcp := &RedisCliProxy{
		listener: listener,
		client:   api.NewMinimalCacheClient(conn),
	}

	for {
		cliConn, err := rcp.listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err.Error())
			continue
		}
		go rcp.serveHandler(cliConn)
	}
}

func (this *RedisCliProxy) serveHandler(conn net.Conn) {
	defer func() {
		conn.Close()
	}()
	rReader := NewRespReader(bufio.NewReader(conn))
	rWriter := NewRespWriter(bufio.NewWriter(conn))
	req, err := rReader.ParseRequest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
	this.handerRequest(rWriter, req)
}

func (this *RedisCliProxy) handerRequest(rWriter *RespWriter, req [][]byte) error {
	var (
		err  error
		full string
	)
	if len(req) == 0 {
		full = ""
	} else {
		tmp := make([]string, 0, len(req))
		for _, value := range req {
			tmp = append(tmp, string(value))
		}
		full = strings.Join(tmp, " ")
	}

	reqx := &api.ExecuteCommandRequest{Command: full}
	resp, err := this.client.ExecuteCommand(context.Background(), reqx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return err
	}
	return this.handleResponse(resp, rWriter)
}

func (this *RedisCliProxy) handleResponse(resp *api.ExecuteCommandResponse, respWrite *RespWriter) error {
	switch resp.Reply {
	case api.OkCommandReply:
		respWrite.FlushString("Ok")
	case api.NilCommandReply:
		respWrite.FlushString("Nil Reply.")
	case api.StringCommandReply:
		respWrite.FlushString(fmt.Sprintf("(string) %s", resp.Item))
	case api.ErrCommandReply:
		respWrite.FlushString(fmt.Sprintf("(error) %s", resp.Item))
	case api.SliceCommandReply:
		var items = make([]interface{}, 0)
		for _, value := range resp.Items {
			items = append(items, value)
		}
		respWrite.FlushArray(items)
	default:
		respWrite.FlushString("not support command")
	}
	return respWrite.Flush()
}
