package nodes

import (
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
)

//go:generate go run declextractor/main.go -DeclName=^InstanceType.+ -SrcPackage=github.com/aws/aws-sdk-go/service/ec2 -DstPackage=nodes -VarName=InstanceTypes -VarType=[]string -DstTemplate=types.tmpl -DstFile=instancetypes.go
//go:generate go run declextractor/main.go -DeclName=^VolumeType.+   -SrcPackage=github.com/aws/aws-sdk-go/service/ec2 -DstPackage=nodes -VarName=VolumeTypes   -VarType=[]string -DstTemplate=types.tmpl -DstFile=volumetypes.go

// obtained from kops/upup/pkg/fi/cloudup/awsup/machine_types.go
var kopsHandledPrefixes = []string{
	"t2.",
	"m3.",
	"m4.",
	"c3.",
	"c4.",
	"cc2.",
	"cg1.",
	"cr1.",
	"d2.",
	"g2.",
	"hi1.",
	"i2.",
	"i3.",
	"r3.",
	"x1.",
	"r4.",
	"p2.",
}

var kopsProhibitedInstTypes = []string{
	ec2.InstanceTypeT2Nano,
	ec2.InstanceTypeT2Micro,
	ec2.InstanceTypeT2Small,
}

var handledTypes []string

func init() {
	handledTypes = make([]string, 0, len(InstanceTypes))
	for _, mType := range InstanceTypes {
		if checkStrInSlice(mType, kopsProhibitedInstTypes) {
			continue
		}

		for _, prefix := range kopsHandledPrefixes {
			if strings.HasPrefix(mType, prefix) {
				handledTypes = append(handledTypes, mType)
				continue
			}
		}
	}
}

// GetNodeTypes returns a list of types possible to be used with create call
func GetNodeTypes() []string {
	return handledTypes
}

// GetVolumeTypes returns a list of types possible to be used with create call
func GetVolumeTypes() []string {
	return VolumeTypes
}

func checkStrInSlice(str string, slc []string) bool {
	for _, chk := range slc {
		if str == chk {
			return true
		}
	}
	return false
}
