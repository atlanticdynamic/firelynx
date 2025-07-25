// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: settings/v1alpha1/apps/v1/echo.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EchoApp struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Response text to echo back to the caller
	// env_interpolation: yes
	Response      *string `protobuf:"bytes,1,opt,name=response" json:"response,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EchoApp) Reset() {
	*x = EchoApp{}
	mi := &file_settings_v1alpha1_apps_v1_echo_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EchoApp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EchoApp) ProtoMessage() {}

func (x *EchoApp) ProtoReflect() protoreflect.Message {
	mi := &file_settings_v1alpha1_apps_v1_echo_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EchoApp.ProtoReflect.Descriptor instead.
func (*EchoApp) Descriptor() ([]byte, []int) {
	return file_settings_v1alpha1_apps_v1_echo_proto_rawDescGZIP(), []int{0}
}

func (x *EchoApp) GetResponse() string {
	if x != nil && x.Response != nil {
		return *x.Response
	}
	return ""
}

var File_settings_v1alpha1_apps_v1_echo_proto protoreflect.FileDescriptor

const file_settings_v1alpha1_apps_v1_echo_proto_rawDesc = "" +
	"\n" +
	"$settings/v1alpha1/apps/v1/echo.proto\x12\x19settings.v1alpha1.apps.v1\"%\n" +
	"\aEchoApp\x12\x1a\n" +
	"\bresponse\x18\x01 \x01(\tR\bresponseBCZAgithub.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1b\beditionsp\xe8\a"

var (
	file_settings_v1alpha1_apps_v1_echo_proto_rawDescOnce sync.Once
	file_settings_v1alpha1_apps_v1_echo_proto_rawDescData []byte
)

func file_settings_v1alpha1_apps_v1_echo_proto_rawDescGZIP() []byte {
	file_settings_v1alpha1_apps_v1_echo_proto_rawDescOnce.Do(func() {
		file_settings_v1alpha1_apps_v1_echo_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_settings_v1alpha1_apps_v1_echo_proto_rawDesc), len(file_settings_v1alpha1_apps_v1_echo_proto_rawDesc)))
	})
	return file_settings_v1alpha1_apps_v1_echo_proto_rawDescData
}

var file_settings_v1alpha1_apps_v1_echo_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_settings_v1alpha1_apps_v1_echo_proto_goTypes = []any{
	(*EchoApp)(nil), // 0: settings.v1alpha1.apps.v1.EchoApp
}
var file_settings_v1alpha1_apps_v1_echo_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_settings_v1alpha1_apps_v1_echo_proto_init() }
func file_settings_v1alpha1_apps_v1_echo_proto_init() {
	if File_settings_v1alpha1_apps_v1_echo_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_settings_v1alpha1_apps_v1_echo_proto_rawDesc), len(file_settings_v1alpha1_apps_v1_echo_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_settings_v1alpha1_apps_v1_echo_proto_goTypes,
		DependencyIndexes: file_settings_v1alpha1_apps_v1_echo_proto_depIdxs,
		MessageInfos:      file_settings_v1alpha1_apps_v1_echo_proto_msgTypes,
	}.Build()
	File_settings_v1alpha1_apps_v1_echo_proto = out.File
	file_settings_v1alpha1_apps_v1_echo_proto_goTypes = nil
	file_settings_v1alpha1_apps_v1_echo_proto_depIdxs = nil
}
