package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	scalekit "github.com/scalekit-inc/scalekit-sdk-go/v2"
	connectionsv1 "github.com/scalekit-inc/scalekit-sdk-go/v2/pkg/grpc/scalekit/v1/connections"
	directoriesv1 "github.com/scalekit-inc/scalekit-sdk-go/v2/pkg/grpc/scalekit/v1/directories"
	"google.golang.org/protobuf/proto"
)

type stepConstructor func(name string, config map[string]any) (sdk.StepInstance, error)

var stepRegistry = map[string]stepConstructor{
	"step.scalekit_auth_provider_describe": newAuthProviderDescribeStep,
	"step.scalekit_connection_create":      newScalekitStep(scalekitConnectionCreate),
	"step.scalekit_connection_get":         newScalekitStep(scalekitConnectionGet),
	"step.scalekit_connection_list":        newScalekitStep(scalekitConnectionList),
	"step.scalekit_connection_enable":      newScalekitStep(scalekitConnectionEnable),
	"step.scalekit_connection_disable":     newScalekitStep(scalekitConnectionDisable),
	"step.scalekit_connection_delete":      newScalekitStep(scalekitConnectionDelete),
	"step.scalekit_directory_create":       newScalekitStep(scalekitDirectoryCreate),
	"step.scalekit_directory_get":          newScalekitStep(scalekitDirectoryGet),
	"step.scalekit_directory_list":         newScalekitStep(scalekitDirectoryList),
	"step.scalekit_directory_enable":       newScalekitStep(scalekitDirectoryEnable),
	"step.scalekit_directory_disable":      newScalekitStep(scalekitDirectoryDisable),
	"step.scalekit_directory_delete":       newScalekitStep(scalekitDirectoryDelete),
	"step.scalekit_directory_user_list":    newScalekitStep(scalekitDirectoryUserList),
	"step.scalekit_directory_group_list":   newScalekitStep(scalekitDirectoryGroupList),
}

func allStepTypes() []string {
	return []string{
		"step.scalekit_auth_provider_describe",
		"step.scalekit_connection_create",
		"step.scalekit_connection_get",
		"step.scalekit_connection_list",
		"step.scalekit_connection_enable",
		"step.scalekit_connection_disable",
		"step.scalekit_connection_delete",
		"step.scalekit_directory_create",
		"step.scalekit_directory_get",
		"step.scalekit_directory_list",
		"step.scalekit_directory_enable",
		"step.scalekit_directory_disable",
		"step.scalekit_directory_delete",
		"step.scalekit_directory_user_list",
		"step.scalekit_directory_group_list",
	}
}

func createStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	constructor, ok := stepRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("scalekit plugin: unknown step type %q", typeName)
	}
	return constructor(name, config)
}

type scalekitHandler func(context.Context, *ScalekitClient, map[string]any) (map[string]any, error)

type scalekitStep struct {
	name       string
	moduleName string
	handler    scalekitHandler
}

func newScalekitStep(handler scalekitHandler) stepConstructor {
	return func(name string, config map[string]any) (sdk.StepInstance, error) {
		moduleName := stringValue(config, "module")
		if moduleName == "" {
			moduleName = "scalekit"
		}
		return &scalekitStep{name: name, moduleName: moduleName, handler: handler}, nil
	}
}

func (s *scalekitStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _, config map[string]any) (*sdk.StepResult, error) {
	client, ok := GetClient(s.moduleName)
	if !ok {
		return &sdk.StepResult{Output: map[string]any{"error": "scalekit client not found: " + s.moduleName}}, nil
	}
	output, err := s.handler(ctx, client, mergeMaps(config, current))
	if err != nil {
		return &sdk.StepResult{Output: errResult(err)}, nil
	}
	return &sdk.StepResult{Output: output}, nil
}

func scalekitConnectionCreate(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, err := requiredID(values, "organization_id", "organizationId", "org_id", "orgId")
	if err != nil {
		return nil, err
	}
	var body connectionsv1.CreateConnection
	if err := decodeProtoBody(values, "connection", &body); err != nil {
		return nil, err
	}
	resp, err := client.SDK.Connection().CreateConnection(ctx, orgID, &body)
	return protoResult("connection", resp, err)
}

func scalekitConnectionGet(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "connection_id", "connectionId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Connection().GetConnection(ctx, orgID, id)
	return protoResult("connection", resp, err)
}

func scalekitConnectionList(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	if domain := stringValue(values, "domain"); domain != "" {
		resp, err := client.SDK.Connection().ListConnectionsByDomain(ctx, domain)
		return protoResult("connections", resp, err)
	}
	orgID, err := requiredID(values, "organization_id", "organizationId", "org_id", "orgId")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Connection().ListConnections(ctx, orgID)
	return protoResult("connections", resp, err)
}

func scalekitConnectionEnable(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "connection_id", "connectionId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Connection().EnableConnection(ctx, orgID, id)
	return protoResult("connection", resp, err)
}

func scalekitConnectionDisable(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "connection_id", "connectionId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Connection().DisableConnection(ctx, orgID, id)
	return protoResult("connection", resp, err)
}

func scalekitConnectionDelete(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "connection_id", "connectionId", "id")
	if err != nil {
		return nil, err
	}
	if err := client.SDK.Connection().DeleteConnection(ctx, orgID, id); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "connection_id": id}, nil
}

func scalekitDirectoryCreate(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, err := requiredID(values, "organization_id", "organizationId", "org_id", "orgId")
	if err != nil {
		return nil, err
	}
	var body directoriesv1.CreateDirectory
	if err := decodeProtoBody(values, "directory", &body); err != nil {
		return nil, err
	}
	resp, err := client.SDK.Directory().CreateDirectory(ctx, orgID, &body)
	return protoResult("directory", resp, err)
}

func scalekitDirectoryGet(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Directory().GetDirectory(ctx, orgID, id)
	return protoResult("directory", resp, err)
}

func scalekitDirectoryList(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, err := requiredID(values, "organization_id", "organizationId", "org_id", "orgId")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Directory().ListDirectories(ctx, orgID)
	return protoResult("directories", resp, err)
}

func scalekitDirectoryEnable(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Directory().EnableDirectory(ctx, orgID, id)
	return protoResult("directory", resp, err)
}

func scalekitDirectoryDisable(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	resp, err := client.SDK.Directory().DisableDirectory(ctx, orgID, id)
	return protoResult("directory", resp, err)
}

func scalekitDirectoryDelete(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, id, err := orgAndID(values, "directory_id", "directoryId", "id")
	if err != nil {
		return nil, err
	}
	if err := client.SDK.Directory().DeleteDirectory(ctx, orgID, id); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "directory_id": id}, nil
}

func scalekitDirectoryUserList(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, dirID, err := orgAndID(values, "directory_id", "directoryId")
	if err != nil {
		return nil, err
	}
	options := &scalekit.ListDirectoryUsersOptions{
		PageSize:  uint32(intValue(values, "page_size", 0)),
		PageToken: stringValue(values, "page_token"),
	}
	resp, err := client.SDK.Directory().ListDirectoryUsers(ctx, orgID, dirID, options)
	return protoResult("users", resp, err)
}

func scalekitDirectoryGroupList(ctx context.Context, client *ScalekitClient, values map[string]any) (map[string]any, error) {
	orgID, dirID, err := orgAndID(values, "directory_id", "directoryId")
	if err != nil {
		return nil, err
	}
	options := &scalekit.ListDirectoryGroupsOptions{
		PageSize:  uint32(intValue(values, "page_size", 0)),
		PageToken: stringValue(values, "page_token"),
	}
	resp, err := client.SDK.Directory().ListDirectoryGroups(ctx, orgID, dirID, options)
	return protoResult("groups", resp, err)
}

func orgAndID(values map[string]any, idKeys ...string) (string, string, error) {
	orgID, err := requiredID(values, "organization_id", "organizationId", "org_id", "orgId")
	if err != nil {
		return "", "", err
	}
	id, err := requiredID(values, idKeys...)
	if err != nil {
		return "", "", err
	}
	return orgID, id, nil
}

func requiredID(values map[string]any, keys ...string) (string, error) {
	id := firstNonEmpty(values, keys...)
	if id == "" {
		return "", fmt.Errorf("%s is required", keys[0])
	}
	return id, nil
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func decodeProtoBody(values map[string]any, key string, target proto.Message) error {
	source := values
	if payload := mapValue(values, key); payload != nil {
		source = payload
	}
	return mapToProtoMessageUntyped(source, target)
}

func protoResult(key string, value proto.Message, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	return protoMessageToMap(key, value)
}
