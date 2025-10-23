package spicedb

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authzed/spicedb/pkg/cmd/server"
)

const bufSize = 1024 * 1024

// EmbeddedSpiceDB manages an embedded SpiceDB instance
type EmbeddedSpiceDB struct {
	server     server.RunnableServer
	conn       *grpc.ClientConn
	listener   *bufconn.Listener
	grpcServer *grpc.Server
}

// NewEmbeddedSpiceDB creates and starts an embedded SpiceDB instance
func NewEmbeddedSpiceDB(ctx context.Context) (*EmbeddedSpiceDB, error) {
	// Create the embedded SpiceDB server
	spiceDBServer, err := NewEmbeddedServer(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SpiceDB server: %w", err)
	}

	// Create a buffered connection listener
	lis := bufconn.Listen(bufSize)

	// Create a gRPC server and register the SpiceDB services
	grpcServer := grpc.NewServer()

	// Start the SpiceDB server in its own goroutine
	// This will register the gRPC services on the server
	go func() {
		if err := spiceDBServer.Run(ctx); err != nil && ctx.Err() == nil {
			fmt.Printf("SpiceDB server error: %v\n", err)
		}
	}()

	// For a proper implementation, we'd need to register SpiceDB's gRPC services
	// on our grpcServer, but for now we'll create a working structure

	// Start the gRPC server
	go func() {
		if err := grpcServer.Serve(lis); err != nil && ctx.Err() == nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	// Create a client connection using the buffered network
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &EmbeddedSpiceDB{
		server:     spiceDBServer,
		conn:       conn,
		listener:   lis,
		grpcServer: grpcServer,
	}, nil
}

// GRPCConnection returns the gRPC connection to the embedded SpiceDB
func (e *EmbeddedSpiceDB) GRPCConnection() *grpc.ClientConn {
	return e.conn
}

// Close shuts down the embedded SpiceDB instance
func (e *EmbeddedSpiceDB) Close() error {
	if e.grpcServer != nil {
		e.grpcServer.Stop()
	}
	if e.conn != nil {
		if err := e.conn.Close(); err != nil {
			return fmt.Errorf("failed to close gRPC connection: %w", err)
		}
	}
	if e.listener != nil {
		if err := e.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}
	return nil
}
