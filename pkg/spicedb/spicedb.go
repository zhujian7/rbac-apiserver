package spicedb

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/spicedb/pkg/cmd/datastore"
	"github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/cmd/util"
)

//go:embed bootstrap.yaml
var bootstrap []byte

// EmbeddedSpiceDB manages an embedded SpiceDB instance
type EmbeddedSpiceDB struct {
	Server            server.RunnableServer
	conn              *grpc.ClientConn
	PermissionsClient v1.PermissionsServiceClient
	SchemaClient      v1.SchemaServiceClient
}

// NewEmbeddedSpiceDB creates and starts an embedded SpiceDB instance
func NewEmbeddedSpiceDB(ctx context.Context) (*EmbeddedSpiceDB, error) {
	// Create the embedded SpiceDB server
	spiceDBServer, err := newEmbeddedServer(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SpiceDB server: %w", err)
	}

	conn, err := spiceDBServer.GRPCDialContext(ctx, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("unable to open gRPC connection with embedded SpiceDB: %w", err)
	}

	return &EmbeddedSpiceDB{
		Server:            spiceDBServer,
		conn:              conn,
		PermissionsClient: v1.NewPermissionsServiceClient(conn),
		SchemaClient:      v1.NewSchemaServiceClient(conn),
	}, nil
}

// GRPCConnection returns the gRPC connection to the embedded SpiceDB
func (e *EmbeddedSpiceDB) GRPCConnection() *grpc.ClientConn {
	return e.conn
}

// Close shuts down the embedded SpiceDB instance
func (e *EmbeddedSpiceDB) Close() error {
	if e.conn != nil {
		if err := e.conn.Close(); err != nil {
			return fmt.Errorf("failed to close gRPC connection: %w", err)
		}
	}
	return nil
}

// newEmbeddedServer creates a new embedded SpiceDB server instance
// for use within the rbac-apiserver.
func newEmbeddedServer(ctx context.Context, bootstrapFilePath string, bootstrapContent map[string][]byte) (server.RunnableServer, error) {
	bootstrapOption := datastore.SetBootstrapFileContents(map[string][]byte{"schema": bootstrap})
	if len(bootstrapContent) > 0 {
		bootstrapOption = datastore.SetBootstrapFileContents(bootstrapContent)
	} else if len(bootstrapFilePath) > 0 {
		bootstrapOption = datastore.SetBootstrapFiles([]string{bootstrapFilePath})
	}

	return server.NewConfigWithOptionsAndDefaults(
		// Configure gRPC server with buffered network for embedded use
		server.WithGRPCServer(util.GRPCServerConfig{
			Network:    util.BufferedNetwork,
			Enabled:    true,
			BufferSize: 10 * humanize.MiByte,
		}),
		// Disable dispatch server (not needed for embedded use)
		server.WithDispatchServer(util.GRPCServerConfig{Enabled: false}),
		server.WithDispatchUpstreamAddr(""),
		server.WithHTTPGatewayUpstreamAddr(""),
		// Configure dispatch and relationship limits
		server.WithDispatchMaxDepth(50),
		server.WithMaximumUpdatesPerWrite(1000),
		server.WithMaximumPreconditionCount(1000),
		server.WithMaxCaveatContextSize(1000000),
		server.WithMaxRelationshipContextSize(1000000),
		server.WithSchemaPrefixesRequired(false),
		// Disable HTTP services (we only need gRPC for embedded use)
		server.WithHTTPGateway(util.HTTPServerConfig{HTTPEnabled: false}),
		server.WithMetricsAPI(util.HTTPServerConfig{HTTPEnabled: false}),
		// Disable telemetry and metrics for embedded mode
		server.WithSilentlyDisableTelemetry(true),
		server.WithDispatchClusterMetricsEnabled(false),
		server.WithDispatchClientMetricsEnabled(false),
		// Disable caching to simplify embedded deployment
		server.WithDispatchCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		server.WithNamespaceCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		server.WithClusterDispatchCacheConfig(server.CacheConfig{Enabled: false, Metrics: false}),
		// Enable experimental features that might be useful
		server.WithEnableExperimentalRelationshipExpiration(true),
		// Configure in-memory datastore
		server.WithDatastoreConfig(
			*datastore.NewConfigWithOptionsAndDefaults().WithOptions(
				datastore.WithEngine(datastore.MemoryEngine),
				bootstrapOption,
				datastore.WithRequestHedgingEnabled(false),
				datastore.WithGCWindow(24*time.Hour),
			)),
		// No authentication for embedded mode (handled by the API server)
		server.WithGRPCAuthFunc(func(ctx context.Context) (context.Context, error) { return ctx, nil }),
	).Complete(ctx)
}
