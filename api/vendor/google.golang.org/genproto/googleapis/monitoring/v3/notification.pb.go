// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/monitoring/v3/notification.proto

package monitoring // import "google.golang.org/genproto/googleapis/monitoring/v3"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"
import _ "google.golang.org/genproto/googleapis/api/annotations"
import label "google.golang.org/genproto/googleapis/api/label"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Indicates whether the channel has been verified or not. It is illegal
// to specify this field in a
// [`CreateNotificationChannel`][google.monitoring.v3.NotificationChannelService.CreateNotificationChannel]
// or an
// [`UpdateNotificationChannel`][google.monitoring.v3.NotificationChannelService.UpdateNotificationChannel]
// operation.
type NotificationChannel_VerificationStatus int32

const (
	// Sentinel value used to indicate that the state is unknown, omitted, or
	// is not applicable (as in the case of channels that neither support
	// nor require verification in order to function).
	NotificationChannel_VERIFICATION_STATUS_UNSPECIFIED NotificationChannel_VerificationStatus = 0
	// The channel has yet to be verified and requires verification to function.
	// Note that this state also applies to the case where the verification
	// process has been initiated by sending a verification code but where
	// the verification code has not been submitted to complete the process.
	NotificationChannel_UNVERIFIED NotificationChannel_VerificationStatus = 1
	// It has been proven that notifications can be received on this
	// notification channel and that someone on the project has access
	// to messages that are delivered to that channel.
	NotificationChannel_VERIFIED NotificationChannel_VerificationStatus = 2
)

var NotificationChannel_VerificationStatus_name = map[int32]string{
	0: "VERIFICATION_STATUS_UNSPECIFIED",
	1: "UNVERIFIED",
	2: "VERIFIED",
}
var NotificationChannel_VerificationStatus_value = map[string]int32{
	"VERIFICATION_STATUS_UNSPECIFIED": 0,
	"UNVERIFIED":                      1,
	"VERIFIED":                        2,
}

func (x NotificationChannel_VerificationStatus) String() string {
	return proto.EnumName(NotificationChannel_VerificationStatus_name, int32(x))
}
func (NotificationChannel_VerificationStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_notification_5449d40305a71b45, []int{1, 0}
}

// A description of a notification channel. The descriptor includes
// the properties of the channel and the set of labels or fields that
// must be specified to configure channels of a given type.
type NotificationChannelDescriptor struct {
	// The full REST resource name for this descriptor. The syntax is:
	//
	//     projects/[PROJECT_ID]/notificationChannelDescriptors/[TYPE]
	//
	// In the above, `[TYPE]` is the value of the `type` field.
	Name string `protobuf:"bytes,6,opt,name=name,proto3" json:"name,omitempty"`
	// The type of notification channel, such as "email", "sms", etc.
	// Notification channel types are globally unique.
	Type string `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	// A human-readable name for the notification channel type.  This
	// form of the name is suitable for a user interface.
	DisplayName string `protobuf:"bytes,2,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	// A human-readable description of the notification channel
	// type. The description may include a description of the properties
	// of the channel and pointers to external documentation.
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	// The set of labels that must be defined to identify a particular
	// channel of the corresponding type. Each label includes a
	// description for how that field should be populated.
	Labels []*label.LabelDescriptor `protobuf:"bytes,4,rep,name=labels,proto3" json:"labels,omitempty"`
	// The tiers that support this notification channel; the project service tier
	// must be one of the supported_tiers.
	SupportedTiers       []ServiceTier `protobuf:"varint,5,rep,packed,name=supported_tiers,json=supportedTiers,proto3,enum=google.monitoring.v3.ServiceTier" json:"supported_tiers,omitempty"` // Deprecated: Do not use.
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *NotificationChannelDescriptor) Reset()         { *m = NotificationChannelDescriptor{} }
func (m *NotificationChannelDescriptor) String() string { return proto.CompactTextString(m) }
func (*NotificationChannelDescriptor) ProtoMessage()    {}
func (*NotificationChannelDescriptor) Descriptor() ([]byte, []int) {
	return fileDescriptor_notification_5449d40305a71b45, []int{0}
}
func (m *NotificationChannelDescriptor) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NotificationChannelDescriptor.Unmarshal(m, b)
}
func (m *NotificationChannelDescriptor) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NotificationChannelDescriptor.Marshal(b, m, deterministic)
}
func (dst *NotificationChannelDescriptor) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NotificationChannelDescriptor.Merge(dst, src)
}
func (m *NotificationChannelDescriptor) XXX_Size() int {
	return xxx_messageInfo_NotificationChannelDescriptor.Size(m)
}
func (m *NotificationChannelDescriptor) XXX_DiscardUnknown() {
	xxx_messageInfo_NotificationChannelDescriptor.DiscardUnknown(m)
}

var xxx_messageInfo_NotificationChannelDescriptor proto.InternalMessageInfo

func (m *NotificationChannelDescriptor) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *NotificationChannelDescriptor) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *NotificationChannelDescriptor) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *NotificationChannelDescriptor) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *NotificationChannelDescriptor) GetLabels() []*label.LabelDescriptor {
	if m != nil {
		return m.Labels
	}
	return nil
}

// Deprecated: Do not use.
func (m *NotificationChannelDescriptor) GetSupportedTiers() []ServiceTier {
	if m != nil {
		return m.SupportedTiers
	}
	return nil
}

// A `NotificationChannel` is a medium through which an alert is
// delivered when a policy violation is detected. Examples of channels
// include email, SMS, and third-party messaging applications. Fields
// containing sensitive information like authentication tokens or
// contact info are only partially populated on retrieval.
type NotificationChannel struct {
	// The type of the notification channel. This field matches the
	// value of the [NotificationChannelDescriptor.type][google.monitoring.v3.NotificationChannelDescriptor.type] field.
	Type string `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	// The full REST resource name for this channel. The syntax is:
	//
	//     projects/[PROJECT_ID]/notificationChannels/[CHANNEL_ID]
	//
	// The `[CHANNEL_ID]` is automatically assigned by the server on creation.
	Name string `protobuf:"bytes,6,opt,name=name,proto3" json:"name,omitempty"`
	// An optional human-readable name for this notification channel. It is
	// recommended that you specify a non-empty and unique name in order to
	// make it easier to identify the channels in your project, though this is
	// not enforced. The display name is limited to 512 Unicode characters.
	DisplayName string `protobuf:"bytes,3,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	// An optional human-readable description of this notification channel. This
	// description may provide additional details, beyond the display
	// name, for the channel. This may not exceeed 1024 Unicode characters.
	Description string `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	// Configuration fields that define the channel and its behavior. The
	// permissible and required labels are specified in the
	// [NotificationChannelDescriptor.labels][google.monitoring.v3.NotificationChannelDescriptor.labels] of the
	// `NotificationChannelDescriptor` corresponding to the `type` field.
	Labels map[string]string `protobuf:"bytes,5,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// User-supplied key/value data that does not need to conform to
	// the corresponding `NotificationChannelDescriptor`'s schema, unlike
	// the `labels` field. This field is intended to be used for organizing
	// and identifying the `NotificationChannel` objects.
	//
	// The field can contain up to 64 entries. Each key and value is limited to
	// 63 Unicode characters or 128 bytes, whichever is smaller. Labels and
	// values can contain only lowercase letters, numerals, underscores, and
	// dashes. Keys must begin with a letter.
	UserLabels map[string]string `protobuf:"bytes,8,rep,name=user_labels,json=userLabels,proto3" json:"user_labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Indicates whether this channel has been verified or not. On a
	// [`ListNotificationChannels`][google.monitoring.v3.NotificationChannelService.ListNotificationChannels]
	// or
	// [`GetNotificationChannel`][google.monitoring.v3.NotificationChannelService.GetNotificationChannel]
	// operation, this field is expected to be populated.
	//
	// If the value is `UNVERIFIED`, then it indicates that the channel is
	// non-functioning (it both requires verification and lacks verification);
	// otherwise, it is assumed that the channel works.
	//
	// If the channel is neither `VERIFIED` nor `UNVERIFIED`, it implies that
	// the channel is of a type that does not require verification or that
	// this specific channel has been exempted from verification because it was
	// created prior to verification being required for channels of this type.
	//
	// This field cannot be modified using a standard
	// [`UpdateNotificationChannel`][google.monitoring.v3.NotificationChannelService.UpdateNotificationChannel]
	// operation. To change the value of this field, you must call
	// [`VerifyNotificationChannel`][google.monitoring.v3.NotificationChannelService.VerifyNotificationChannel].
	VerificationStatus NotificationChannel_VerificationStatus `protobuf:"varint,9,opt,name=verification_status,json=verificationStatus,proto3,enum=google.monitoring.v3.NotificationChannel_VerificationStatus" json:"verification_status,omitempty"`
	// Whether notifications are forwarded to the described channel. This makes
	// it possible to disable delivery of notifications to a particular channel
	// without removing the channel from all alerting policies that reference
	// the channel. This is a more convenient approach when the change is
	// temporary and you want to receive notifications from the same set
	// of alerting policies on the channel at some point in the future.
	Enabled              *wrappers.BoolValue `protobuf:"bytes,11,opt,name=enabled,proto3" json:"enabled,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *NotificationChannel) Reset()         { *m = NotificationChannel{} }
func (m *NotificationChannel) String() string { return proto.CompactTextString(m) }
func (*NotificationChannel) ProtoMessage()    {}
func (*NotificationChannel) Descriptor() ([]byte, []int) {
	return fileDescriptor_notification_5449d40305a71b45, []int{1}
}
func (m *NotificationChannel) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NotificationChannel.Unmarshal(m, b)
}
func (m *NotificationChannel) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NotificationChannel.Marshal(b, m, deterministic)
}
func (dst *NotificationChannel) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NotificationChannel.Merge(dst, src)
}
func (m *NotificationChannel) XXX_Size() int {
	return xxx_messageInfo_NotificationChannel.Size(m)
}
func (m *NotificationChannel) XXX_DiscardUnknown() {
	xxx_messageInfo_NotificationChannel.DiscardUnknown(m)
}

var xxx_messageInfo_NotificationChannel proto.InternalMessageInfo

func (m *NotificationChannel) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *NotificationChannel) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *NotificationChannel) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *NotificationChannel) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *NotificationChannel) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *NotificationChannel) GetUserLabels() map[string]string {
	if m != nil {
		return m.UserLabels
	}
	return nil
}

func (m *NotificationChannel) GetVerificationStatus() NotificationChannel_VerificationStatus {
	if m != nil {
		return m.VerificationStatus
	}
	return NotificationChannel_VERIFICATION_STATUS_UNSPECIFIED
}

func (m *NotificationChannel) GetEnabled() *wrappers.BoolValue {
	if m != nil {
		return m.Enabled
	}
	return nil
}

func init() {
	proto.RegisterType((*NotificationChannelDescriptor)(nil), "google.monitoring.v3.NotificationChannelDescriptor")
	proto.RegisterType((*NotificationChannel)(nil), "google.monitoring.v3.NotificationChannel")
	proto.RegisterMapType((map[string]string)(nil), "google.monitoring.v3.NotificationChannel.LabelsEntry")
	proto.RegisterMapType((map[string]string)(nil), "google.monitoring.v3.NotificationChannel.UserLabelsEntry")
	proto.RegisterEnum("google.monitoring.v3.NotificationChannel_VerificationStatus", NotificationChannel_VerificationStatus_name, NotificationChannel_VerificationStatus_value)
}

func init() {
	proto.RegisterFile("google/monitoring/v3/notification.proto", fileDescriptor_notification_5449d40305a71b45)
}

var fileDescriptor_notification_5449d40305a71b45 = []byte{
	// 602 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x54, 0x6d, 0x6b, 0xdb, 0x3c,
	0x14, 0x7d, 0x9c, 0x34, 0x7d, 0x5a, 0xb9, 0xa4, 0x9d, 0x5a, 0x86, 0xf1, 0xde, 0xd2, 0xee, 0xc3,
	0xf2, 0xc9, 0x86, 0x64, 0x83, 0x75, 0x6f, 0xd0, 0xa4, 0xe9, 0x08, 0xac, 0x59, 0xc9, 0xdb, 0xa0,
	0x14, 0x82, 0x92, 0xa8, 0x9e, 0x98, 0x2d, 0x19, 0x49, 0xf6, 0xc8, 0xcf, 0xd8, 0x8f, 0xd8, 0x87,
	0xed, 0xa7, 0xec, 0x57, 0x0d, 0xcb, 0x8a, 0xed, 0xb5, 0x86, 0x75, 0xdf, 0x74, 0xcf, 0x3d, 0xe7,
	0xdc, 0x7b, 0x4f, 0x4c, 0xc0, 0x33, 0x8f, 0x31, 0xcf, 0xc7, 0x6e, 0xc0, 0x28, 0x91, 0x8c, 0x13,
	0xea, 0xb9, 0x71, 0xdb, 0xa5, 0x4c, 0x92, 0x6b, 0xb2, 0x40, 0x92, 0x30, 0xea, 0x84, 0x9c, 0x49,
	0x06, 0x0f, 0x52, 0xa2, 0x93, 0x13, 0x9d, 0xb8, 0x6d, 0x3f, 0xd4, 0x72, 0x14, 0x12, 0x17, 0x51,
	0xca, 0xa4, 0x92, 0x88, 0x54, 0x63, 0xdf, 0x2f, 0x74, 0x7d, 0x34, 0xc7, 0xbe, 0xc6, 0x0f, 0x4b,
	0x87, 0x2e, 0x58, 0x10, 0xac, 0xc7, 0xd9, 0x8f, 0x35, 0x45, 0x55, 0xf3, 0xe8, 0xda, 0xfd, 0xca,
	0x51, 0x18, 0x62, 0xae, 0xad, 0x8f, 0xbe, 0x55, 0xc0, 0xa3, 0x41, 0x61, 0xcb, 0xee, 0x67, 0x44,
	0x29, 0xf6, 0x4f, 0xb1, 0x58, 0x70, 0x12, 0x4a, 0xc6, 0x21, 0x04, 0x1b, 0x14, 0x05, 0xd8, 0xda,
	0x6c, 0x18, 0xcd, 0xed, 0xa1, 0x7a, 0x27, 0x98, 0x5c, 0x85, 0xd8, 0x32, 0x52, 0x2c, 0x79, 0xc3,
	0x43, 0xb0, 0xb3, 0x24, 0x22, 0xf4, 0xd1, 0x6a, 0xa6, 0xf8, 0x15, 0xd5, 0x33, 0x35, 0x36, 0x48,
	0x64, 0x0d, 0x60, 0x2e, 0xb5, 0x31, 0x61, 0xd4, 0xaa, 0x6a, 0x46, 0x0e, 0xc1, 0x36, 0xd8, 0x54,
	0x07, 0x0a, 0x6b, 0xa3, 0x51, 0x6d, 0x9a, 0xad, 0x07, 0x8e, 0x8e, 0x0b, 0x85, 0xc4, 0xf9, 0x90,
	0x74, 0xf2, 0xcd, 0x86, 0x9a, 0x0a, 0x07, 0x60, 0x57, 0x44, 0x61, 0xc8, 0xb8, 0xc4, 0xcb, 0x99,
	0x24, 0x98, 0x0b, 0xab, 0xd6, 0xa8, 0x36, 0xeb, 0xad, 0x43, 0xa7, 0x2c, 0x6c, 0x67, 0x84, 0x79,
	0x4c, 0x16, 0x78, 0x4c, 0x30, 0xef, 0x54, 0x2c, 0x63, 0x58, 0xcf, 0xd4, 0x09, 0x24, 0x8e, 0xbe,
	0xd7, 0xc0, 0x7e, 0x49, 0x26, 0xa5, 0x57, 0x97, 0xa5, 0x73, 0x33, 0x89, 0xea, 0x5f, 0x93, 0xd8,
	0xb8, 0x9d, 0xc4, 0x79, 0x96, 0x44, 0x4d, 0x25, 0xf1, 0xa2, 0xfc, 0x96, 0x92, 0x3d, 0xd3, 0x9c,
	0x44, 0x8f, 0x4a, 0xbe, 0xca, 0x32, 0xba, 0x04, 0x66, 0x24, 0x30, 0x9f, 0x69, 0xcf, 0x2d, 0xe5,
	0x79, 0x7c, 0x77, 0xcf, 0x89, 0xc0, 0xbc, 0xe8, 0x0b, 0xa2, 0x0c, 0x80, 0x01, 0xd8, 0x8f, 0x31,
	0xcf, 0x24, 0x33, 0x21, 0x91, 0x8c, 0x84, 0xb5, 0xdd, 0x30, 0x9a, 0xf5, 0xd6, 0x9b, 0xbb, 0xcf,
	0x98, 0x16, 0x4c, 0x46, 0xca, 0x63, 0x08, 0xe3, 0x5b, 0x18, 0x7c, 0x0e, 0xfe, 0xc7, 0x14, 0xcd,
	0x7d, 0xbc, 0xb4, 0xcc, 0x86, 0xd1, 0x34, 0x5b, 0xf6, 0x7a, 0xc4, 0xfa, 0x23, 0x77, 0x3a, 0x8c,
	0xf9, 0x53, 0xe4, 0x47, 0x78, 0xb8, 0xa6, 0xda, 0xc7, 0xc0, 0x2c, 0xec, 0x0f, 0xf7, 0x40, 0xf5,
	0x0b, 0x5e, 0xe9, 0x9f, 0x32, 0x79, 0xc2, 0x03, 0x50, 0x8b, 0x13, 0x89, 0xfe, 0x70, 0xd3, 0xe2,
	0x55, 0xe5, 0xa5, 0x61, 0xbf, 0x05, 0xbb, 0x37, 0xce, 0xff, 0x17, 0xf9, 0xd1, 0x27, 0x00, 0x6f,
	0x5f, 0x06, 0x9f, 0x82, 0x27, 0xd3, 0xde, 0xb0, 0x7f, 0xd6, 0xef, 0x9e, 0x8c, 0xfb, 0x1f, 0x07,
	0xb3, 0xd1, 0xf8, 0x64, 0x3c, 0x19, 0xcd, 0x26, 0x83, 0xd1, 0x45, 0xaf, 0xdb, 0x3f, 0xeb, 0xf7,
	0x4e, 0xf7, 0xfe, 0x83, 0x75, 0x00, 0x26, 0x83, 0x94, 0xd6, 0x3b, 0xdd, 0x33, 0xe0, 0x0e, 0xd8,
	0xca, 0xaa, 0x4a, 0xe7, 0x87, 0x01, 0xac, 0x05, 0x0b, 0x4a, 0x03, 0xee, 0xdc, 0x2b, 0x26, 0x7c,
	0x91, 0x04, 0x73, 0x61, 0x5c, 0xbe, 0xd3, 0x54, 0x8f, 0xf9, 0x88, 0x7a, 0x0e, 0xe3, 0x9e, 0xeb,
	0x61, 0xaa, 0x62, 0x73, 0xd3, 0x16, 0x0a, 0x89, 0xf8, 0xf3, 0xff, 0xe4, 0x75, 0x5e, 0xfd, 0xac,
	0xd8, 0xef, 0x53, 0x83, 0xae, 0xcf, 0xa2, 0xa5, 0x73, 0x9e, 0x4f, 0x9c, 0xb6, 0x7f, 0xad, 0x9b,
	0x57, 0xaa, 0x79, 0x95, 0x37, 0xaf, 0xa6, 0xed, 0xf9, 0xa6, 0x1a, 0xd2, 0xfe, 0x1d, 0x00, 0x00,
	0xff, 0xff, 0xf7, 0x1b, 0x09, 0x21, 0x28, 0x05, 0x00, 0x00,
}
