package internal_gengo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var allExtensions []*extensionInfo

func genMessageOptions(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo) {
	if m.Desc.Options() == nil {
		return
	}

	// TODO(mawei): 目前没有用处
	m.Desc.Options().ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		return true
	})

	opt := m.Desc.Options().ProtoReflect()
	parseUnknow(opt.GetUnknown(), func(number protowire.Number, v interface{}) {
		genCustomMethod(g, f, m, opt.Descriptor(), number, v)
	})
}

func parseUnknow(unknow []byte, f func(number protowire.Number, v interface{})) {
	for len(unknow) > 0 {
		i, t, n := protowire.ConsumeTag(unknow)
		unknow = unknow[n:]
		switch t {
		case protowire.VarintType:
			var v uint64
			v, n = protowire.ConsumeVarint(unknow)
			f(i, v)
		case protowire.BytesType:
			var v []byte
			v, n = protowire.ConsumeBytes(unknow)
			f(i, string(v))
		case protowire.Fixed32Type:
			var v uint32
			v, n = protowire.ConsumeFixed32(unknow)
			f(i, v)
		case protowire.Fixed64Type:
			var v uint64
			v, n = protowire.ConsumeFixed64(unknow)
			f(i, v)
		// case protowire.StartGroupType:
		// 	var v []byte
		// 	v, n = protowire.ConsumeGroup(i, unknow)
		// 	parseUnknow(v, f)
		default:
			panic(fmt.Sprintf("prototext: error parsing unknown field wire type: %v", t))
		}
		unknow = unknow[n:]
	}
}
func genCustomMethod(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo, desc protoreflect.MessageDescriptor, field protoreflect.FieldNumber, val interface{}) {
	msgExtInfo := getExtensionInfo(desc, field)
	if msgExtInfo == nil {
		return
	}

	if v, ok := val.(string); ok {
		val = "\"" + v + "\""
	}
	goType, _ := fieldGoType(g, f, msgExtInfo.Extension)

	fieldOption := msgExtInfo.Desc.Options().ProtoReflect()
	if fieldOption != nil {
		parseUnknow(fieldOption.GetUnknown(), func(number protowire.Number, v interface{}) {
			fieldExtInfo := getExtensionInfo(fieldOption.Descriptor(), number)
			if fieldExtInfo == nil {
				return
			}
			switch strings.ToLower(fieldExtInfo.GoName) {
			case "type":
				goType = v.(string)
			}
		})
	}

	g.P("func (x ", m.GoIdent, ")", msgExtInfo.GoName, "()", goType, "{ return ", val, "}")
	g.P()
}

func getExtensionInfo(desc protoreflect.Descriptor, field protoreflect.FieldNumber) *extensionInfo {
	for _, ext := range allExtensions {
		if desc.FullName() != ext.Extendee.Desc.FullName() || field != ext.Desc.Number() {
			continue
		}
		return ext
	}
	return nil
}

// TODO:不能识别嵌套的 extension
func GenAllExtensionInfo(files []*protogen.File) {
	for _, f := range files {
		for _, ext := range f.Extensions {
			allExtensions = append(allExtensions, newExtensionInfo(nil, ext))
		}
	}
}

type debugWrite struct {
	b bytes.Buffer
}

func (d *debugWrite) Write(format string, v ...interface{}) {
	fmt.Fprintf(&d.b, format+"\n", v...)
}

func (d *debugWrite) Flush() {
	ioutil.WriteFile("data.txt", d.b.Bytes(), 0664)
}
