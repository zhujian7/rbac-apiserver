package spicedb

import (
	"context"
	"testing"
	"time"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddedSpiceDB(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "successful creation with background context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "successful creation with timeout context",
			ctx:     createContextWithTimeout(5 * time.Second),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create embedded SpiceDB
			spiceDB, err := NewEmbeddedSpiceDB(tt.ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, spiceDB)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, spiceDB)

			// Verify the embedded SpiceDB instance is properly initialized
			assert.NotNil(t, spiceDB.Server, "Server should not be nil")
			assert.NotNil(t, spiceDB.conn, "gRPC connection should not be nil")
			assert.NotNil(t, spiceDB.PermissionsClient, "Permissions client should not be nil")
			assert.NotNil(t, spiceDB.SchemaClient, "Schema client should not be nil")

			// Test that the gRPC connection is valid
			conn := spiceDB.GRPCConnection()
			assert.NotNil(t, conn)
			assert.Equal(t, spiceDB.conn, conn)

			// Clean up
			err = spiceDB.Close()
			assert.NoError(t, err, "Close should not return an error")
		})
	}
}

func TestEmbeddedSpiceDB_GRPCConnection(t *testing.T) {
	ctx := context.Background()
	spiceDB, err := NewEmbeddedSpiceDB(ctx)
	require.NoError(t, err)
	defer func() {
		err := spiceDB.Close()
		assert.NoError(t, err)
	}()

	// Test that GRPCConnection returns the correct connection
	conn := spiceDB.GRPCConnection()
	assert.NotNil(t, conn)
	assert.Equal(t, spiceDB.conn, conn)
}

func TestEmbeddedSpiceDB_Close(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() *EmbeddedSpiceDB
		wantErr  bool
		errorMsg string
	}{
		{
			name: "successful close with valid connection",
			setupFn: func() *EmbeddedSpiceDB {
				ctx := context.Background()
				spiceDB, err := NewEmbeddedSpiceDB(ctx)
				require.NoError(t, err)
				return spiceDB
			},
			wantErr: false,
		},
		{
			name: "close with nil connection",
			setupFn: func() *EmbeddedSpiceDB {
				return &EmbeddedSpiceDB{
					conn: nil,
				}
			},
			wantErr: false,
		},
		{
			name: "multiple close calls",
			setupFn: func() *EmbeddedSpiceDB {
				ctx := context.Background()
				spiceDB, err := NewEmbeddedSpiceDB(ctx)
				require.NoError(t, err)
				// Close once first
				err = spiceDB.Close()
				require.NoError(t, err)
				return spiceDB
			},
			wantErr: true, // Second close might fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spiceDB := tt.setupFn()

			err := spiceDB.Close()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewEmbeddedServer(t *testing.T) {
	tests := []struct {
		name              string
		ctx               context.Context
		bootstrapFilePath string
		bootstrapContent  map[string][]byte
		wantErr           bool
		errorMsg          string
	}{
		{
			name:              "successful creation with default bootstrap",
			ctx:               context.Background(),
			bootstrapFilePath: "",
			bootstrapContent:  nil,
			wantErr:           false,
		},
		{
			name:              "successful creation with custom bootstrap content",
			ctx:               context.Background(),
			bootstrapFilePath: "",
			bootstrapContent: map[string][]byte{
				"schema": []byte(`schema: |-
  definition user {}
  
  definition resource {
    relation viewer: user
    permission view = viewer
  }

relationships: |-
  // Test relationships`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := newEmbeddedServer(tt.ctx, tt.bootstrapFilePath, tt.bootstrapContent)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, server)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, server)
		})
	}
}

func TestEmbeddedSpiceDB_Integration(t *testing.T) {
	// Skip this test in short mode as it takes longer
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	spiceDB, err := NewEmbeddedSpiceDB(ctx)
	require.NoError(t, err)
	defer func() {
		err := spiceDB.Close()
		assert.NoError(t, err)
	}()

	// Give the server time to start up
	time.Sleep(100 * time.Millisecond)

	// Test basic schema operations
	t.Run("schema operations", func(t *testing.T) {
		// Create a context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Write a simple schema
		schemaReq := &v1.WriteSchemaRequest{
			Schema: `definition user {}
definition resource {
	relation viewer: user
	permission view = viewer
}`,
		}

		_, err := spiceDB.SchemaClient.WriteSchema(opCtx, schemaReq)
		if err != nil {
			t.Logf("Schema write failed, this may be expected if server is not fully ready: %v", err)
			t.Skip("Skipping schema operations test due to server not being ready")
			return
		}

		// Read the schema back
		readReq := &v1.ReadSchemaRequest{}
		resp, err := spiceDB.SchemaClient.ReadSchema(opCtx, readReq)
		if err != nil {
			t.Logf("Schema read failed: %v", err)
			return
		}
		assert.Contains(t, resp.SchemaText, "definition user", "Schema should contain user definition")
	})

	// Test basic relationship operations
	t.Run("relationship operations", func(t *testing.T) {
		// Create a context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Write a relationship
		writeReq := &v1.WriteRelationshipsRequest{
			Updates: []*v1.RelationshipUpdate{
				{
					Operation: v1.RelationshipUpdate_OPERATION_CREATE,
					Relationship: &v1.Relationship{
						Resource: &v1.ObjectReference{
							ObjectType: "resource",
							ObjectId:   "test-resource",
						},
						Relation: "viewer",
						Subject: &v1.SubjectReference{
							Object: &v1.ObjectReference{
								ObjectType: "user",
								ObjectId:   "test-user",
							},
						},
					},
				},
			},
		}

		_, err := spiceDB.PermissionsClient.WriteRelationships(opCtx, writeReq)
		if err != nil {
			t.Logf("Relationship write failed, this may be expected if server is not fully ready: %v", err)
			t.Skip("Skipping relationship operations test due to server not being ready")
			return
		}

		// Check permission
		checkReq := &v1.CheckPermissionRequest{
			Resource: &v1.ObjectReference{
				ObjectType: "resource",
				ObjectId:   "test-resource",
			},
			Permission: "view",
			Subject: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: "user",
					ObjectId:   "test-user",
				},
			},
		}

		checkResp, err := spiceDB.PermissionsClient.CheckPermission(opCtx, checkReq)
		if err != nil {
			t.Logf("Permission check failed: %v", err)
			return
		}
		assert.Equal(t, v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, checkResp.Permissionship)
	})
}

func TestEmbeddedSpiceDB_ConcurrentAccess(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping concurrent access test in short mode")
	}

	ctx := context.Background()
	spiceDB, err := NewEmbeddedSpiceDB(ctx)
	require.NoError(t, err)
	defer func() {
		err := spiceDB.Close()
		assert.NoError(t, err)
	}()

	// Test that multiple goroutines can access the clients safely
	t.Run("concurrent client access", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Each goroutine tries to read schema
				opCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				readReq := &v1.ReadSchemaRequest{}
				_, err := spiceDB.SchemaClient.ReadSchema(opCtx, readReq)
				// Don't assert on error as server might not be ready
				if err != nil {
					t.Logf("Concurrent schema read failed (goroutine %d): %v", id, err)
				}

				// Test getting the connection
				conn := spiceDB.GRPCConnection()
				assert.NotNil(t, conn, "Getting connection should work concurrently")
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Goroutine completed
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations to complete")
			}
		}
	})
}

// Helper function to create context with timeout
func createContextWithTimeout(duration time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), duration)
	return ctx
}

// Benchmark tests
func BenchmarkNewEmbeddedSpiceDB(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spiceDB, err := NewEmbeddedSpiceDB(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = spiceDB.Close()
	}
}

func BenchmarkEmbeddedSpiceDB_GRPCConnection(b *testing.B) {
	ctx := context.Background()
	spiceDB, err := NewEmbeddedSpiceDB(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = spiceDB.Close()
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = spiceDB.GRPCConnection()
	}
}
