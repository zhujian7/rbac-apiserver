package spicedb

import (
	"context"
	_ "embed"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/authzed/spicedb/pkg/cmd/datastore"
	"github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/cmd/util"
)

//go:embed bootstrap.yaml
var bootstrap []byte

// NewEmbeddedServer creates a new embedded SpiceDB server instance
// for use within the rbac-apiserver.
func NewEmbeddedServer(ctx context.Context, bootstrapFilePath string, bootstrapContent map[string][]byte) (server.RunnableServer, error) {
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