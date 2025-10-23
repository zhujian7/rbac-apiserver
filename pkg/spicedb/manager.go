package spicedb

import (
	"context"
	"fmt"
	"sync"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"google.golang.org/grpc"
)

// Manager provides access to the embedded SpiceDB instance
type Manager struct {
	mu              sync.RWMutex
	embeddedSpiceDB *EmbeddedSpiceDB
	permissionsClient v1.PermissionsServiceClient
	schemaClient      v1.SchemaServiceClient
}

var globalManager *Manager
var once sync.Once

// GetManager returns the global SpiceDB manager instance
func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{}
	})
	return globalManager
}

// Initialize sets up the SpiceDB manager with the embedded instance
func (m *Manager) Initialize(ctx context.Context, embeddedDB *EmbeddedSpiceDB) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embeddedSpiceDB = embeddedDB
	
	// Create gRPC clients
	conn := embeddedDB.GRPCConnection()
	m.permissionsClient = v1.NewPermissionsServiceClient(conn)
	m.schemaClient = v1.NewSchemaServiceClient(conn)

	return nil
}

// PermissionsClient returns the permissions service client
func (m *Manager) PermissionsClient() v1.PermissionsServiceClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.permissionsClient
}

// SchemaClient returns the schema service client
func (m *Manager) SchemaClient() v1.SchemaServiceClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.schemaClient
}

// Connection returns the raw gRPC connection
func (m *Manager) Connection() *grpc.ClientConn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.embeddedSpiceDB == nil {
		return nil
	}
	return m.embeddedSpiceDB.GRPCConnection()
}

// CheckPermission is a convenience method for checking permissions
func (m *Manager) CheckPermission(ctx context.Context, resource, permission, subject string) (bool, error) {
	client := m.PermissionsClient()
	if client == nil {
		return false, fmt.Errorf("SpiceDB client not initialized")
	}

	resp, err := client.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Resource: &v1.ObjectReference{
			ObjectType: "resource",
			ObjectId:   resource,
		},
		Permission: permission,
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: "user",
				ObjectId:   subject,
			},
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, nil
}

// WriteRelationships is a convenience method for writing relationships
func (m *Manager) WriteRelationships(ctx context.Context, updates []*v1.RelationshipUpdate) error {
	client := m.PermissionsClient()
	if client == nil {
		return fmt.Errorf("SpiceDB client not initialized")
	}

	_, err := client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: updates,
	})
	return err
}

// Close shuts down the SpiceDB manager
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.embeddedSpiceDB != nil {
		return m.embeddedSpiceDB.Close()
	}
	return nil
}