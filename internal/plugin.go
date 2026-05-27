package internal

import (
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-scalekit/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

var Version = "0.0.0"

type scalekitPlugin struct{}

func NewScalekitPlugin() sdk.PluginProvider {
	return &scalekitPlugin{}
}

func (p *scalekitPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-scalekit",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Scalekit enterprise SSO and Directory Sync provider plugin",
	}
}

func (p *scalekitPlugin) ModuleTypes() []string {
	return []string{"scalekit.provider"}
}

func (p *scalekitPlugin) TypedModuleTypes() []string {
	return p.ModuleTypes()
}

func (p *scalekitPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "scalekit.provider":
		return newScalekitModule(name, config)
	default:
		return nil, fmt.Errorf("scalekit plugin: unknown module type %q", typeName)
	}
}

func (p *scalekitPlugin) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	if typeName != "scalekit.provider" {
		return nil, fmt.Errorf("scalekit plugin: unknown typed module type %q", typeName)
	}
	factory := sdk.NewTypedModuleFactory(typeName, &contracts.ProviderConfig{}, func(name string, cfg *contracts.ProviderConfig) (sdk.ModuleInstance, error) {
		return newScalekitModule(name, typedModuleConfig(cfg))
	})
	return factory.CreateTypedModule(typeName, name, config)
}

func (p *scalekitPlugin) StepTypes() []string {
	return allStepTypes()
}

func (p *scalekitPlugin) TypedStepTypes() []string {
	return p.StepTypes()
}

func (p *scalekitPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	return createStep(typeName, name, config)
}

func (p *scalekitPlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	if _, ok := stepRegistry[typeName]; !ok {
		return nil, fmt.Errorf("%w: step type %q", sdk.ErrTypedContractNotHandled, typeName)
	}
	if typeName == "step.scalekit_auth_provider_describe" {
		return sdk.NewTypedStepFactory(typeName, &contracts.AuthProviderDescribeConfig{}, &contracts.AuthProviderDescribeInput{}, typedAuthProviderDescribe).CreateTypedStep(typeName, name, config)
	}
	return sdk.NewTypedStepFactory(typeName, &contracts.ScalekitStepConfig{}, &contracts.ScalekitStepInput{}, typedStepHandler(typeName)).CreateTypedStep(typeName, name, config)
}

func (p *scalekitPlugin) ContractRegistry() *pb.ContractRegistry {
	return contractRegistry
}

var contractRegistry = &pb.ContractRegistry{
	FileDescriptorSet: &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
			protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_scalekit_proto),
		},
	},
	Contracts: contractDescriptors(),
}

func contractDescriptors() []*pb.ContractDescriptor {
	descriptors := []*pb.ContractDescriptor{
		moduleContract("scalekit.provider", "ProviderConfig"),
	}
	for _, stepType := range allStepTypes() {
		if stepType == "step.scalekit_auth_provider_describe" {
			descriptors = append(descriptors, stepContract(stepType, "AuthProviderDescribeConfig", "AuthProviderDescribeInput", "AuthProviderDescribeOutput"))
			continue
		}
		descriptors = append(descriptors, stepContract(stepType, "ScalekitStepConfig", "ScalekitStepInput", "ScalekitStepOutput"))
	}
	return descriptors
}

func moduleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.scalekit.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: pkg + configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	const pkg = "workflow.plugins.scalekit.v1."
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: pkg + configMessage,
		InputMessage:  pkg + inputMessage,
		OutputMessage: pkg + outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}
