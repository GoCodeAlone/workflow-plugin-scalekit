package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-scalekit/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	scalekit "github.com/scalekit-inc/scalekit-sdk-go/v2"
	connectionsv1 "github.com/scalekit-inc/scalekit-sdk-go/v2/pkg/grpc/scalekit/v1/connections"
	directoriesv1 "github.com/scalekit-inc/scalekit-sdk-go/v2/pkg/grpc/scalekit/v1/directories"
)

func TestModuleInitRegistersScalekitClient(t *testing.T) {
	module, err := newScalekitModule("scalekit-test", map[string]any{
		"environmentUrl": "https://scalekit.example.test",
		"clientId":       "client-id",
		"clientSecret":   "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	client, ok := GetClient("scalekit-test")
	if !ok || client == nil || client.SDK == nil {
		t.Fatal("expected registered SDK client")
	}
	if client.EnvironmentURL != "https://scalekit.example.test" {
		t.Fatalf("environment url = %q", client.EnvironmentURL)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetClient("scalekit-test"); ok {
		t.Fatal("expected client to be unregistered")
	}
}

func TestModuleInitRequiresConfig(t *testing.T) {
	module, err := newScalekitModule("scalekit-test", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing environment_url error")
	}
	module, err = newScalekitModule("scalekit-test", map[string]any{"environmentUrl": "https://scalekit.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing client_id error")
	}
}

func TestContractRegistryIncludesStrictProtoDescriptors(t *testing.T) {
	provider := NewScalekitPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	registry := provider.ContractRegistry()
	if registry == nil || registry.GetFileDescriptorSet() == nil {
		t.Fatal("missing contract registry file descriptors")
	}
	contractsByType := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.GetContracts() {
		switch contract.GetKind() {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByType["module:"+contract.GetModuleType()] = contract
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByType["step:"+contract.GetStepType()] = contract
		}
	}
	module := contractsByType["module:scalekit.provider"]
	if module == nil || module.GetConfigMessage() != "workflow.plugins.scalekit.v1.ProviderConfig" {
		t.Fatalf("unexpected module contract: %#v", module)
	}
	for _, stepType := range allStepTypes() {
		contract := contractsByType["step:"+stepType]
		if contract == nil {
			t.Fatalf("missing step contract %s", stepType)
		}
		if contract.GetMode() != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v", stepType, contract.GetMode())
		}
	}
}

func TestDescriptorAdvertisesOnlyScalekitCapabilities(t *testing.T) {
	step, err := newAuthProviderDescribeStep("describe", map[string]any{"environmentUrl": "https://scalekit.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := result.Output["providers"].([]map[string]any)[0]
	categories := stringSet(provider["categories"].([]string))
	for _, category := range []string{"enterprise_sso", "directory_sync"} {
		if !categories[category] {
			t.Fatalf("missing category %q", category)
		}
	}
	for _, absent := range []string{"identity_management", "authentication_method", "oauth2_oidc", "rbac", "mfa"} {
		if categories[absent] {
			t.Fatalf("descriptor must not advertise %s", absent)
		}
	}
	capabilities := provider["capabilities"].([]map[string]any)
	if len(capabilities) != 3 {
		t.Fatalf("capability count = %d", len(capabilities))
	}
	for _, capability := range capabilities {
		if capability["supported"] != true {
			t.Fatalf("%s supported = %#v", capability["key"], capability["supported"])
		}
	}
}

func TestTypedDescriptor(t *testing.T) {
	result, err := typedAuthProviderDescribe(context.Background(), sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]{
		Config: &contracts.AuthProviderDescribeConfig{ProviderId: "scalekit-admin"},
		Input:  &contracts.AuthProviderDescribeInput{EnvironmentUrl: "https://scalekit.example.test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == nil || len(result.Output.GetProviders()) != 1 {
		t.Fatalf("providers = %#v", result.Output)
	}
	if result.Output.GetProviders()[0].GetId() != "scalekit-admin" {
		t.Fatalf("provider id = %q", result.Output.GetProviders()[0].GetId())
	}
}

func TestConnectionStepsUseOfficialSDK(t *testing.T) {
	connection := &fakeConnection{}
	registerFakeClient(t, &fakeScalekitSDK{connection: connection, directory: &fakeDirectory{}})

	result := executeStep(t, "step.scalekit_connection_create", map[string]any{
		"organization_id": "org-1",
		"connection": map[string]any{
			"provider":     "IDP_SIMULATOR",
			"type":         "OIDC",
			"provider_key": "primary",
		},
	})
	if connection.createOrgID != "org-1" || connection.createBody.GetProviderKey() != "primary" {
		t.Fatalf("create call = org %q body %#v", connection.createOrgID, connection.createBody)
	}
	if nestedID(result, "connection", "connection") != "conn-1" {
		t.Fatalf("create output = %#v", result)
	}

	result = executeStep(t, "step.scalekit_connection_list", map[string]any{"domain": "example.com"})
	if connection.listDomain != "example.com" || nestedListID(result, "connections", "connections") != "conn-domain" {
		t.Fatalf("domain list call/output = domain %q output %#v", connection.listDomain, result)
	}

	result = executeStep(t, "step.scalekit_connection_enable", map[string]any{"organization_id": "org-1", "connection_id": "conn-1"})
	if connection.enabledID != "conn-1" || boolField(result, "connection", "enabled") != true {
		t.Fatalf("enable call/output = id %q output %#v", connection.enabledID, result)
	}
}

func TestDirectoryStepsUseOfficialSDK(t *testing.T) {
	directory := &fakeDirectory{}
	registerFakeClient(t, &fakeScalekitSDK{connection: &fakeConnection{}, directory: directory})

	result := executeStep(t, "step.scalekit_directory_create", map[string]any{
		"organization_id": "org-1",
		"directory": map[string]any{
			"directory_type":     "SCIM",
			"directory_provider": "OKTA",
		},
	})
	if directory.createOrgID != "org-1" || directory.createBody.GetDirectoryProvider() != directoriesv1.DirectoryProvider_OKTA {
		t.Fatalf("create call = org %q body %#v", directory.createOrgID, directory.createBody)
	}
	if nestedID(result, "directory", "directory") != "dir-1" {
		t.Fatalf("create output = %#v", result)
	}

	result = executeStep(t, "step.scalekit_directory_user_list", map[string]any{"organization_id": "org-1", "directory_id": "dir-1", "page_size": 25})
	if directory.usersDirectoryID != "dir-1" || directory.usersOptions.PageSize != 25 || nestedListID(result, "users", "users") != "user-1" {
		t.Fatalf("users call/output = dir %q options %#v output %#v", directory.usersDirectoryID, directory.usersOptions, result)
	}

	result = executeStep(t, "step.scalekit_directory_group_list", map[string]any{"organization_id": "org-1", "directory_id": "dir-1"})
	if directory.groupsDirectoryID != "dir-1" || nestedListID(result, "groups", "groups") != "group-1" {
		t.Fatalf("groups call/output = dir %q output %#v", directory.groupsDirectoryID, result)
	}
}

func TestMissingClientReturnsErrorOutput(t *testing.T) {
	UnregisterClient("missing")
	step, err := createStep("step.scalekit_connection_get", "get", map[string]any{"module": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"organization_id": "org-1", "connection_id": "conn-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] == "" {
		t.Fatalf("expected error output, got %#v", result.Output)
	}
}

func registerFakeClient(t *testing.T, fake *fakeScalekitSDK) {
	t.Helper()
	RegisterClient("scalekit-test", &ScalekitClient{SDK: fake, EnvironmentURL: "https://scalekit.example.test"})
	t.Cleanup(func() { UnregisterClient("scalekit-test") })
}

func executeStep(t *testing.T, stepType string, values map[string]any) map[string]any {
	t.Helper()
	step, err := createStep(stepType, "test", map[string]any{"module": "scalekit-test"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, values)
	if err != nil {
		t.Fatal(err)
	}
	if errText, _ := result.Output["error"].(string); errText != "" {
		t.Fatalf("step returned error output: %s", errText)
	}
	return result.Output
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	return set
}

func nestedID(values map[string]any, envelopeKey, resourceKey string) string {
	envelope, _ := values[envelopeKey].(map[string]any)
	resource, _ := envelope[resourceKey].(map[string]any)
	id, _ := resource["id"].(string)
	return id
}

func nestedListID(values map[string]any, envelopeKey, listKey string) string {
	envelope, _ := values[envelopeKey].(map[string]any)
	list, _ := envelope[listKey].([]any)
	if len(list) == 0 {
		return ""
	}
	item, _ := list[0].(map[string]any)
	id, _ := item["id"].(string)
	return id
}

func boolField(values map[string]any, envelopeKey, field string) bool {
	envelope, _ := values[envelopeKey].(map[string]any)
	value, _ := envelope[field].(bool)
	return value
}

type fakeScalekitSDK struct {
	connection scalekit.Connection
	directory  scalekit.Directory
}

func (f *fakeScalekitSDK) Connection() scalekit.Connection { return f.connection }
func (f *fakeScalekitSDK) Directory() scalekit.Directory   { return f.directory }

type fakeConnection struct {
	createOrgID string
	createBody  *connectionsv1.CreateConnection
	listDomain  string
	enabledID   string
}

func (f *fakeConnection) CreateConnection(_ context.Context, organizationID string, body *connectionsv1.CreateConnection) (*scalekit.CreateConnectionResponse, error) {
	f.createOrgID = organizationID
	f.createBody = body
	org := organizationID
	return &connectionsv1.CreateConnectionResponse{Connection: &connectionsv1.Connection{Id: "conn-1", OrganizationId: &org, ProviderKey: body.GetProviderKey()}}, nil
}

func (f *fakeConnection) GetConnection(_ context.Context, organizationID, id string) (*scalekit.GetConnectionResponse, error) {
	org := organizationID
	return &connectionsv1.GetConnectionResponse{Connection: &connectionsv1.Connection{Id: id, OrganizationId: &org}}, nil
}

func (f *fakeConnection) ListConnectionsByDomain(_ context.Context, domain string) (*scalekit.ListConnectionsResponse, error) {
	f.listDomain = domain
	return &connectionsv1.ListConnectionsResponse{Connections: []*connectionsv1.ListConnection{{Id: "conn-domain"}}}, nil
}

func (f *fakeConnection) ListConnections(_ context.Context, _ string) (*scalekit.ListConnectionsResponse, error) {
	return &connectionsv1.ListConnectionsResponse{Connections: []*connectionsv1.ListConnection{{Id: "conn-org"}}}, nil
}

func (f *fakeConnection) EnableConnection(_ context.Context, _, id string) (*scalekit.ToggleConnectionResponse, error) {
	f.enabledID = id
	return &connectionsv1.ToggleConnectionResponse{Enabled: true}, nil
}

func (f *fakeConnection) DisableConnection(_ context.Context, _, _ string) (*scalekit.ToggleConnectionResponse, error) {
	return &connectionsv1.ToggleConnectionResponse{Enabled: false}, nil
}

func (f *fakeConnection) DeleteConnection(_ context.Context, _, _ string) error { return nil }

type fakeDirectory struct {
	createOrgID       string
	createBody        *directoriesv1.CreateDirectory
	usersDirectoryID  string
	usersOptions      *scalekit.ListDirectoryUsersOptions
	groupsDirectoryID string
}

func (f *fakeDirectory) CreateDirectory(_ context.Context, organizationID string, body *directoriesv1.CreateDirectory) (*scalekit.CreateDirectoryResponse, error) {
	f.createOrgID = organizationID
	f.createBody = body
	return &directoriesv1.CreateDirectoryResponse{Directory: &directoriesv1.Directory{Id: "dir-1", OrganizationId: organizationID, DirectoryProvider: body.GetDirectoryProvider()}}, nil
}

func (f *fakeDirectory) ListDirectories(_ context.Context, organizationID string) (*scalekit.ListDirectoriesResponse, error) {
	return &directoriesv1.ListDirectoriesResponse{Directories: []*directoriesv1.Directory{{Id: "dir-1", OrganizationId: organizationID}}}, nil
}

func (f *fakeDirectory) ListDirectoryUsers(_ context.Context, _ string, directoryID string, options *scalekit.ListDirectoryUsersOptions) (*scalekit.ListDirectoryUsersResponse, error) {
	f.usersDirectoryID = directoryID
	f.usersOptions = options
	return &directoriesv1.ListDirectoryUsersResponse{Users: []*directoriesv1.DirectoryUser{{Id: "user-1"}}}, nil
}

func (f *fakeDirectory) ListDirectoryGroups(_ context.Context, _ string, directoryID string, _ *scalekit.ListDirectoryGroupsOptions) (*scalekit.ListDirectoryGroupsResponse, error) {
	f.groupsDirectoryID = directoryID
	return &directoriesv1.ListDirectoryGroupsResponse{Groups: []*directoriesv1.DirectoryGroup{{Id: "group-1"}}}, nil
}

func (f *fakeDirectory) GetPrimaryDirectoryByOrganizationId(_ context.Context, organizationID string) (*scalekit.GetDirectoryResponse, error) {
	return &directoriesv1.GetDirectoryResponse{Directory: &directoriesv1.Directory{Id: "dir-primary", OrganizationId: organizationID}}, nil
}

func (f *fakeDirectory) EnableDirectory(_ context.Context, _, _ string) (*scalekit.ToggleDirectoryResponse, error) {
	return &directoriesv1.ToggleDirectoryResponse{Enabled: true}, nil
}

func (f *fakeDirectory) DisableDirectory(_ context.Context, _, _ string) (*scalekit.ToggleDirectoryResponse, error) {
	return &directoriesv1.ToggleDirectoryResponse{Enabled: false}, nil
}

func (f *fakeDirectory) GetDirectory(_ context.Context, organizationID, directoryID string) (*scalekit.GetDirectoryResponse, error) {
	return &directoriesv1.GetDirectoryResponse{Directory: &directoriesv1.Directory{Id: directoryID, OrganizationId: organizationID}}, nil
}

func (f *fakeDirectory) DeleteDirectory(_ context.Context, _, _ string) error { return nil }
