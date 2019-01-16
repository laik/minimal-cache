package cluster

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/http"

	"github.com/gogo/gateway"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/laik/minimal-cache/api"
	"github.com/laik/minimal-cache/insecure"
	"github.com/laik/minimal-cache/redis"
	"github.com/rakyll/statik/fs"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (this *RaftNode) redisProxy(grpcAddr string) {
	redis.NewRedisCliProxy(grpcAddr)
}

func (this *RaftNode) grpcServer(grpcAddr string) error {
	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %v", grpcAddr, err)
	}
	this.srv = grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
		grpc.UnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
	)
	api.RegisterMinimalCacheServer(this.srv, this)

	return this.srv.Serve(l)
}

func (this *RaftNode) gwServer(grpcAddr, httpAddr string) error {
	/// gRPC gateway service
	/// See https://github.com/grpc/grpc/blob/master/doc/naming.md
	/// for gRPC naming standard information.
	dialAddr := fmt.Sprintf("passthrough://localhost/%s", grpcAddr)

	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
		grpc.WithBlock(),
	)
	if err != nil {
		this.log.Fatalln("Failed to dial server:", err)
	}

	mux := http.NewServeMux()

	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(
			runtime.MIMEWildcard,
			&gateway.JSONPb{
				EmitDefaults: true,
				Indent:       "  ",
				OrigName:     true,
			},
		),
		// This is necessary to get error details properly,marshalled in unary requests.
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)

	err = api.RegisterMinimalCacheHandler(context.Background(), gwmux, conn)
	if err != nil {
		this.log.Fatalln("Failed to register gateway:", err)
	}

	mux.Handle("/", gwmux)
	err = serveOpenAPI(mux)
	if err != nil {
		this.log.Fatalln("Failed to serve OpenAPI UI")
		return err
	}

	gatewayAddr := fmt.Sprintf("%s", httpAddr)
	this.log.Info("Serving gRPC-Gateway on https://", gatewayAddr)
	this.log.Info("Serving OpenAPI Documentation on https://", gatewayAddr, "/openapi-ui/")

	gwServer := http.Server{
		Addr:      gatewayAddr,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{insecure.Cert}},
		Handler:   mux,
	}

	return gwServer.ListenAndServeTLS("", "")
}

func (this *RaftNode) start(grpcAddr string, httpAddr string) (err error) {
	go this.grpcServer(grpcAddr)

	go this.gwServer(grpcAddr, httpAddr)

	go this.redisProxy(grpcAddr)

	go this.whenLeaderChanged(this.updateLeaderIP, this.controlLeaderConn)

	select {}
}

func serveOpenAPI(mux *http.ServeMux) error {
	mime.AddExtensionType(".svg", "image/svg+xml")

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	// Expose files in static on <host>/openapi-ui
	fileServer := http.FileServer(statikFS)
	prefix := "/openapi-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	return nil
}
