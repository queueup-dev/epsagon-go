// Code generated by protoc-gen-go. DO NOT EDIT.
// source: error_code.proto

package protocol

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ErrorCode int32

const (
	ErrorCode_OK        ErrorCode = 0
	ErrorCode_ERROR     ErrorCode = 1
	ErrorCode_EXCEPTION ErrorCode = 2
)

var ErrorCode_name = map[int32]string{
	0: "OK",
	1: "ERROR",
	2: "EXCEPTION",
}

var ErrorCode_value = map[string]int32{
	"OK":        0,
	"ERROR":     1,
	"EXCEPTION": 2,
}

func (x ErrorCode) String() string {
	return proto.EnumName(ErrorCode_name, int32(x))
}

func (ErrorCode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_c5513ac0a8e17e40, []int{0}
}

func init() {
	proto.RegisterEnum("protocol.ErrorCode", ErrorCode_name, ErrorCode_value)
}

func init() { proto.RegisterFile("error_code.proto", fileDescriptor_c5513ac0a8e17e40) }

var fileDescriptor_c5513ac0a8e17e40 = []byte{
	// 99 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x48, 0x2d, 0x2a, 0xca,
	0x2f, 0x8a, 0x4f, 0xce, 0x4f, 0x49, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x00, 0x53,
	0xc9, 0xf9, 0x39, 0x5a, 0xba, 0x5c, 0x9c, 0xae, 0x20, 0x59, 0xe7, 0xfc, 0x94, 0x54, 0x21, 0x36,
	0x2e, 0x26, 0x7f, 0x6f, 0x01, 0x06, 0x21, 0x4e, 0x2e, 0x56, 0xd7, 0xa0, 0x20, 0xff, 0x20, 0x01,
	0x46, 0x21, 0x5e, 0x2e, 0x4e, 0xd7, 0x08, 0x67, 0xd7, 0x80, 0x10, 0x4f, 0x7f, 0x3f, 0x01, 0xa6,
	0x24, 0x36, 0xb0, 0x46, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xec, 0xf2, 0x6c, 0x0e, 0x53,
	0x00, 0x00, 0x00,
}