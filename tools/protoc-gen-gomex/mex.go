// Copyright 2022 MobiledgeX, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/edgexr/edge-cloud-platform/pkg/gensupport"
	"github.com/edgexr/edge-cloud-platform/pkg/util"
	"github.com/edgexr/edge-cloud-platform/tools/protogen"
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

func RegisterMex() {
	generator.RegisterPlugin(new(mex))
}

func init() {
	generator.RegisterPlugin(new(mex))
}

type mex struct {
	gen                    *generator.Generator
	msgs                   map[string]*descriptor.DescriptorProto
	cudTemplate            *template.Template
	fieldsValTemplate      *template.Template
	enumTemplate           *template.Template
	cacheTemplate          *template.Template
	keysTemplate           *template.Template
	sublistLookupTemplate  *template.Template
	subfieldLookupTemplate *template.Template
	importUtil             bool
	importLog              bool
	importStrings          bool
	importErrors           bool
	importStrconv          bool
	importSort             bool
	importTime             bool
	importCmp              bool
	importReflect          bool
	importJson             bool
	importSync             bool
	importObjstore         bool
	importContext          bool
	importRedis            bool
	firstFile              string
	support                gensupport.PluginSupport
	refData                *gensupport.RefData
	keyMessages            []descriptor.DescriptorProto
	deletePrepareFields    map[string]string
}

func (m *mex) Name() string {
	return "mex"
}

func (m *mex) Init(gen *generator.Generator) {
	m.gen = gen
	m.msgs = make(map[string]*descriptor.DescriptorProto)
	m.deletePrepareFields = make(map[string]string)
	m.cudTemplate = template.Must(template.New("cud").Parse(cudTemplateIn))
	m.fieldsValTemplate = template.Must(template.New("fieldsVal").Parse(fieldsValTemplate))
	m.enumTemplate = template.Must(template.New("enum").Parse(enumTemplateIn))
	m.cacheTemplate = template.Must(template.New("cache").Parse(cacheTemplateIn))
	m.keysTemplate = template.Must(template.New("keys").Parse(keysTemplateIn))
	m.sublistLookupTemplate = template.Must(template.New("sublist").Parse(sublistLookupTemplateIn))
	m.subfieldLookupTemplate = template.Must(template.New("subfield").Parse(subfieldLookupTemplateIn))
	m.support.Init(gen.Request)
	m.firstFile = gensupport.GetFirstFile(gen)
}

// P forwards to g.gen.P
func (m *mex) P(args ...interface{}) {
	m.gen.P(args...)
}

func (m *mex) getAllKeyMessages() {
	for _, file := range m.gen.Request.ProtoFile {
		for _, desc := range file.MessageType {
			if GetObjKey(desc) {
				m.keyMessages = append(m.keyMessages, *desc)
			}
		}
	}
}

func (m *mex) Generate(file *generator.FileDescriptor) {
	m.support.InitFile()
	m.support.SetPbGoPackage(file.GetPackage())
	m.importUtil = false
	m.importLog = false
	m.importStrings = false
	m.importErrors = false
	m.importStrconv = false
	m.importSort = false
	m.importTime = false
	m.importCmp = false
	m.importReflect = false
	m.importJson = false
	m.importSync = false
	m.importObjstore = false
	m.importContext = false
	m.importRedis = false
	if m.firstFile == *file.FileDescriptorProto.Name {
		m.refData = m.support.GatherRefData(m.gen)
		m.checkDeletePrepares()
	}
	for _, desc := range file.Messages() {
		m.generateMessage(file, desc)
	}
	for _, desc := range file.Enums() {
		m.generateEnum(file, desc)
	}
	if len(file.FileDescriptorProto.Service) != 0 {
		for _, service := range file.FileDescriptorProto.Service {
			m.generateService(file, service)
		}
	}

	if m.firstFile == *file.FileDescriptorProto.Name {
		m.P(matchOptions)
		m.P(fieldMap)
		m.generateEnumDecodeHook()
		m.generateShowCheck()
		m.generateAllKeyTags()
		m.generateGetReferences()
		m.importStrings = true
		m.importSort = true
	}
}

func (m *mex) GenerateImports(file *generator.FileDescriptor) {
	hasGenerateCud := false
	fileDeps := make(map[string]struct{})
	for _, dep := range file.Dependency {
		fileDeps[dep] = struct{}{}
	}
	for _, desc := range file.Messages() {
		msg := desc.DescriptorProto
		if GetGenerateCud(msg) {
			hasGenerateCud = true
		}
		m.msgs[*msg.Name] = msg
	}
	if hasGenerateCud {
		m.gen.PrintImport("", "encoding/json")
		m.importObjstore = true
	}
	if hasGenerateCud || m.firstFile == *file.FileDescriptorProto.Name {
		m.gen.PrintImport("", "go.etcd.io/etcd/client/v3/concurrency")
	}
	if m.importObjstore {
		m.gen.PrintImport("", "github.com/edgexr/edge-cloud-platform/pkg/objstore")
	}
	if m.importUtil {
		m.gen.PrintImport("", "github.com/edgexr/edge-cloud-platform/pkg/util")
	}
	if m.importLog {
		m.gen.PrintImport("", "github.com/edgexr/edge-cloud-platform/pkg/log")
	}
	if m.importStrings {
		m.gen.PrintImport("strings", "strings")
	}
	if m.importErrors {
		m.gen.PrintImport("", "errors")
	}
	if m.importStrconv {
		m.gen.PrintImport("", "strconv")
	}
	if m.importJson {
		m.gen.PrintImport("", "encoding/json")
	}
	if m.importSort {
		m.gen.PrintImport("", "sort")
	}
	if m.importTime {
		m.gen.PrintImport("", "time")
	}
	if m.importContext {
		m.gen.PrintImport("context", "context")
	}
	if m.importReflect {
		m.gen.PrintImport("reflect", "reflect")
	}
	if m.importSync {
		m.gen.PrintImport("", "sync")
	}
	if m.importCmp {
		m.gen.PrintImport("", "github.com/google/go-cmp/cmp")
		m.gen.PrintImport("", "github.com/google/go-cmp/cmp/cmpopts")
	}
	if m.importRedis {
		m.gen.PrintImport("", "github.com/go-redis/redis/v8")
	}
	m.support.PrintUsedImportsPlugin(m.gen, fileDeps)
}

func (m *mex) generateEnum(file *generator.FileDescriptor, desc *generator.EnumDescriptor) {
	en := desc.EnumDescriptorProto
	m.P("var ", en.Name, "Strings = []string{")
	for _, val := range en.Value {
		m.P("\"", val.Name, "\",")
	}
	m.P("}")
	m.P()
	// generate bit map for debug levels
	if len(en.Value) <= 64 {
		m.P("const (")
		for ii, val := range en.Value {
			m.P(en.Name, generator.CamelCase(*val.Name), " uint64 = 1 << ", ii)
		}
		m.P(")")
		m.P()
	}
	// generate camel case maps
	fqname := m.support.FQTypeName(m.gen, desc)
	m.P("var ", fqname, "_CamelName = map[int32]string{")
	for _, val := range en.Value {
		m.P("// ", val.Name, " -> ", util.CamelCase(*val.Name))
		m.P(val.Number, ": \"", util.CamelCase(*val.Name), "\",")
	}
	m.P("}")
	m.P("var ", fqname, "_CamelValue = map[string]int32{")
	for _, val := range en.Value {
		m.P("\"", util.CamelCase(*val.Name), "\": ", val.Number, ",")
	}
	m.P("}")
	m.P()

	args := enumTempl{
		Name:         m.support.FQTypeName(m.gen, desc),
		CommonPrefix: gensupport.GetEnumCommonPrefix(en),
	}
	m.enumTemplate.Execute(m.gen.Buffer, args)
	m.importStrconv = true
	m.importJson = true
	m.importReflect = true
	m.importUtil = true
	if len(args.CommonPrefix) > 0 {
		m.importStrings = true
		m.P("var ", en.Name, "CommonPrefix = \"", args.CommonPrefix, "\"")
		m.P()
	}

	if GetVersionHashOpt(en) {
		// Collect all key objects
		m.getAllKeyMessages()
		salt := GetVersionHashSalt(en)
		hashStr := fmt.Sprintf("%x", getKeyVersionHash(m.keyMessages, salt))
		// get latest version field ID
		lastIndex := 0
		for i, _ := range en.Value {
			if i > lastIndex {
				lastIndex = i
			}
		}
		latestVerEnum := en.Value[lastIndex]
		// Generate a hash of all the key messages.
		m.generateVersionString(hashStr, *latestVerEnum.Number)
		// Generate version check code for version message
		validateVersionHash(latestVerEnum, hashStr, file)
	}
}

type enumTempl struct {
	Name         string
	CommonPrefix string
}

var enumTemplateIn = `
func Parse{{.Name}}(data interface{}) ({{.Name}}, error) {
	if val, ok := data.({{.Name}}); ok {
		return val, nil
	} else if str, ok := data.(string); ok {
		val, ok := {{.Name}}_CamelValue[util.CamelCase(str)]
{{- if .CommonPrefix}}
		if !ok {
			// may have omitted common prefix
			val, ok = {{.Name}}_CamelValue["{{.CommonPrefix}}"+util.CamelCase(str)]
		}
{{- end}}
		if !ok {
			// may be int value instead of enum name
			ival, err := strconv.Atoi(str)
			val = int32(ival)
			if err == nil {
				_, ok = {{.Name}}_CamelName[val]
			}
		}
		if !ok {
			return {{.Name}}(0), fmt.Errorf("Invalid {{.Name}} value %q", str)
		}
		return {{.Name}}(val), nil
	} else if ival, ok := data.(int32); ok {
		if _, ok := {{.Name}}_CamelName[ival]; ok {
			return {{.Name}}(ival), nil
		} else {
			return {{.Name}}(0), fmt.Errorf("Invalid {{.Name}} value %d", ival)
		}
	}
	return {{.Name}}(0), fmt.Errorf("Invalid {{.Name}} value %v", data)
}

func (e *{{.Name}}) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	err := unmarshal(&str)
	if err != nil { return err }
	val, err := Parse{{.Name}}(str)
	if err != nil {
		return err
	}
	*e = val
	return nil
}

func (e {{.Name}}) MarshalYAML() (interface{}, error) {
	str := proto.EnumName({{.Name}}_CamelName, int32(e))
{{- if .CommonPrefix}}
	str = strings.TrimPrefix(str, "{{.CommonPrefix}}")
{{- end}}
	return str, nil
}

// custom JSON encoding/decoding
func (e *{{.Name}}) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err == nil {
		val, err := Parse{{.Name}}(str)
		if err != nil {
			return &json.UnmarshalTypeError{
				Value: "string " + str,
				Type: reflect.TypeOf({{.Name}}(0)),
			}
		}
		*e = {{.Name}}(val)
		return nil
	}
	var ival int32
	err = json.Unmarshal(b, &ival)
	if err == nil {
		val, err := Parse{{.Name}}(ival)
		if err == nil {
			*e = val
			return nil
		}
	}
	return &json.UnmarshalTypeError{
		Value: "value " + string(b),
		Type: reflect.TypeOf({{.Name}}(0)),
	}
}

func (e {{.Name}}) MarshalJSON() ([]byte, error) {
	str := proto.EnumName({{.Name}}_CamelName, int32(e))
{{- if .CommonPrefix}}
	str = strings.TrimPrefix(str, "{{.CommonPrefix}}")
{{- end}}
	return json.Marshal(str)
}
`

type MatchType int

const (
	FieldMatch MatchType = iota
	ExactMatch
	IgnoreBackendMatch
)

var matchOptions = `
type MatchOptions struct {
	// Filter will ignore 0 or nil fields on the passed in object
	Filter bool
	// IgnoreBackend will ignore fields that were marked backend in .proto
	IgnoreBackend bool
	// Sort repeated (arrays) of Key objects so matching does not
	// fail due to order.
	SortArrayedKeys bool
}

type MatchOpt func(*MatchOptions)

func MatchFilter() MatchOpt {
	return func(opts *MatchOptions) {
		opts.Filter = true
	}
}

func MatchIgnoreBackend() MatchOpt {
	return func(opts *MatchOptions) {
		opts.IgnoreBackend = true
	}
}

func MatchSortArrayedKeys() MatchOpt {
	return func(opts *MatchOptions) {
		opts.SortArrayedKeys = true
	}
}

func applyMatchOptions(opts *MatchOptions, args ...MatchOpt) {
	for _, f := range args {
		f(opts)
	}
}

`

var fieldMap = `
type FieldMap struct {
	fields map[string]struct{}
}

func MakeFieldMap(fields []string) *FieldMap {
	fmap := &FieldMap{}
	fmap.fields = map[string]struct{}{}
	if fields == nil {
		return fmap
	}
	for _, set := range fields {
		fmap.fields[set] = struct{}{}
	}
	return fmap
}

func NewFieldMap(fields map[string]struct{}) *FieldMap {
	if fields == nil {
		fields = map[string]struct{}{}
	}
	return &FieldMap{
		fields: fields,
	}
}

// Has checks if the key is set. Note that setting
// a parent key implies that all child keys are also set.
func (s *FieldMap) Has(key string) bool {
	// key or parent is specified
	for {
		if _, ok := s.fields[key]; ok {
			return true
		}
		idx := strings.LastIndex(key, ".")
		if idx == -1 {
			break
		}
		key = key[:idx]
	}
	return false
}

// HasOrHasChild checks if the key, or any child
// of the key is set. Note that as with Has(), if
// a parent of the key is set, this returns true.
func (s *FieldMap) HasOrHasChild(key string) bool {
	if s.Has(key) {
		return true
	}
	prefix := key + "."
	for k := range s.fields {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func (s *FieldMap) Set(key string) {
	s.fields[key] = struct{}{}
}

func (s *FieldMap) Clear(key string) {
	delete(s.fields, key)
}

func (s *FieldMap) Fields() []string {
	fields := []string{}
	for k := range s.fields {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return fields
}

func (s *FieldMap) Count() int {
	return len(s.fields)
}

// OptionalSTM is for operations that use either the cache or the store.
type OptionalSTM struct {
	// STM may be nil to force using the cache instead of the store
	stm concurrency.STM
}

// NewOptionalSTM creates a new optional STM for operations that
// use either the cache or the store. Set the stm to force using
// the store, or leave nil to force using the cache.
func NewOptionalSTM(stm concurrency.STM) *OptionalSTM {
	return &OptionalSTM{
		stm: stm,
	}
}

`

func (m *mex) generateFieldMatches(message *descriptor.DescriptorProto, field *descriptor.FieldDescriptorProto) {
	if field.Type == nil {
		return
	}
	backend := GetBackend(field)
	if backend {
		m.P("if !opts.IgnoreBackend {")
	}

	// ignore field if filter was specified and o.name is 0 or nil
	nilval := "0"
	nilCheck := true
	repeated := false
	name := generator.CamelCase(*field.Name)
	if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED ||
		*field.Type == descriptor.FieldDescriptorProto_TYPE_BYTES {
		nilval = "nil"
		repeated = true
	} else {
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			if !gogoproto.IsNullable(field) {
				nilCheck = false
			}
			nilval = "nil"
		case descriptor.FieldDescriptorProto_TYPE_STRING:
			nilval = "\"\""
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			nilval = "false"
		}
	}
	if nilCheck {
		m.P("if !opts.Filter || o.", name, " != ", nilval, " {")
	}
	if nilCheck && nilval == "nil" {
		if repeated {
			m.P("if len(m.", name, ") == 0 && len(o.", name, ") > 0 || len(m.", name, ") > 0 && len(o.", name, ") == 0 {")
		} else {
			m.P("if m.", name, " == nil && o.", name, " != nil || m.", name, " != nil && o.", name, " == nil {")
		}
		m.P("return false")
		m.P("} else if m.", name, " != nil && o.", name, "!= nil {")
	}

	mapType := m.support.GetMapType(m.gen, field)
	oName := ""
	mName := ""
	if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED ||
		*field.Type == descriptor.FieldDescriptorProto_TYPE_BYTES {
		m.P("if !opts.Filter && len(m.", name, ") != len(o.", name, ") {")
		m.P("return false")
		m.P("}")
		if mapType == nil {
			skipMatch := false
			if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
				if GetObjKey(subDesc.DescriptorProto) {
					m.P("if opts.SortArrayedKeys {")
					m.P("sort.Slice(m.", name, ", func(i, j int) bool {")
					m.P("return m.", name, "[i].GetKeyString() < m.", name, "[j].GetKeyString()")
					m.P("})")
					m.P("sort.Slice(o.", name, ", func(i, j int) bool {")
					m.P("return o.", name, "[i].GetKeyString() < o.", name, "[j].GetKeyString()")
					m.P("})")
					m.P("}")
					m.importSort = true
				}
				if !GetGenerateMatches(subDesc.DescriptorProto) {
					skipMatch = true
				}
			}
			if !skipMatch {
				m.P("found := 0")
				m.P("for oIndex, _ := range o.", name, " {")
				m.P("for mIndex, _ := range m.", name, " {")
				oName = name + "[oIndex]"
				mName = name + "[mIndex]"
			}
		} else {
			m.P("for k, _ := range o.", name, " {")
			m.P("_, ok := m.", name, "[k]")
			m.P("if !ok {")
			m.P("return false")
			m.P("}")
			name = name + "[k]"
			field = mapType.ValField
		}
	}
	switch *field.Type {
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		ref := "&"
		if gogoproto.IsNullable(field) {
			ref = ""
		}
		subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
		printedCheck := true
		if *field.TypeName == ".google.protobuf.Timestamp" {
			m.P("if m.", name, ".Seconds != o.", name, ".Seconds || m.", name, ".Nanos != o.", name, ".Nanos {")
		} else if GetGenerateMatches(subDesc.DescriptorProto) {
			if oName != "" && mName != "" {
				m.P("if m.", mName, ".Matches(", ref, "o.", oName, ", fopts...) {")
				m.P("found++")
				m.P("break")
				m.P("}")
				printedCheck = false
			} else {
				m.P("if !m.", name, ".Matches(", ref, "o.", name, ", fopts...) {")
			}
		} else {
			printedCheck = false
		}
		if printedCheck {
			m.P("return false")
			m.P("}")
		}
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		// deprecated in proto3
	default:
		if oName != "" && mName != "" {
			m.P("if o.", oName, " == m.", mName, "{")
			m.P("found++")
			m.P("break")
			m.P("}")
		} else {
			m.P("if o.", name, " != m.", name, "{")
			m.P("return false")
			m.P("}")
		}
	}
	if repeated {
		if oName != "" && mName != "" {
			m.P("}")
			m.P("}")
			m.P("if found != len(o.", name, ") {")
			m.P("return false")
			m.P("}")
		}
		if mapType != nil {
			m.P("}")
		}
	}
	if nilCheck && nilval == "nil" {
		m.P("}")
	}
	if nilCheck {
		m.P("}")
	}
	if backend {
		m.P("}")
	}
}

func (m *mex) getInvalidMethodFields(names []string, subAllInvalidFields bool, desc *generator.Descriptor, method *descriptor.MethodDescriptorProto) {
	message := desc.DescriptorProto
	noconfig := gensupport.GetNoConfig(message, method)
	noconfigMap := make(map[string]string)
	for _, nc := range strings.Split(noconfig, ",") {
		if nc == "" {
			continue
		}
		noconfigMap["."+nc] = "0"
	}
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if keyField := gensupport.GetMessageKey(message); keyField != nil {
			if *keyField.Name == *field.Name {
				continue
			}
		}
		nilval := "0"
		nilcheck := true
		name := generator.CamelCase(*field.Name)
		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED ||
			*field.Type == descriptor.FieldDescriptorProto_TYPE_BYTES {
			nilval = "nil"
		} else {
			switch *field.Type {
			case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
				nilval = "nil"
				if !gogoproto.IsNullable(field) {
					nilcheck = false
				}
			case descriptor.FieldDescriptorProto_TYPE_STRING:
				nilval = "\"\""
			case descriptor.FieldDescriptorProto_TYPE_BOOL:
				nilval = "false"
			}
		}
		fieldName := strings.Join(append(names, name), ".")
		nullableMessage := false
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			if nilcheck {
				if _, ok := noconfigMap[fieldName]; !ok {
					for ncField, _ := range noconfigMap {
						if strings.HasPrefix(ncField, fieldName) {
							m.P("if m", fieldName, " != ", nilval, " {")
							nullableMessage = true
							break
						}
					}
				}
				if _, ok := noconfigMap[fieldName]; ok || subAllInvalidFields {
					m.P("if m", fieldName, " != ", nilval, " {")
					argStr := strings.TrimLeft(fieldName, ".")
					m.P("return fmt.Errorf(\"Invalid field specified: ", argStr, ", this field is only for internal use\")")
					m.P("}")
					continue
				}
			}
			subAllInvalidFields := false
			if _, ok := noconfigMap[fieldName]; ok {
				subAllInvalidFields = true
			}
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.getInvalidMethodFields(append(names, name), subAllInvalidFields, subDesc, method)
			if nullableMessage {
				m.P("}")
			}
		default:
			if _, ok := noconfigMap[fieldName]; ok || subAllInvalidFields {
				m.P("if m", fieldName, " != ", nilval, " {")
				argStr := strings.TrimLeft(fieldName, ".")
				m.P("return fmt.Errorf(\"Invalid field specified: ", argStr, ", this field is only for internal use\")")
				m.P("}")
			}
		}
	}
}

func (m *mex) printCopyInMakeArray(name string, desc *generator.Descriptor, field *descriptor.FieldDescriptorProto) {
	mapType := m.support.GetMapType(m.gen, field)
	if mapType != nil {
		valType := mapType.ValType
		if mapType.ValIsMessage && gogoproto.IsNullable(field) {
			valType = "*" + valType
		}
		m.P("m.", name, " = make(map[", mapType.KeyType, "]", valType, ")")
	}
}

func (m *mex) getFieldDesc(field *descriptor.FieldDescriptorProto) *generator.Descriptor {
	obj := m.gen.ObjectNamed(field.GetTypeName())
	if obj == nil {
		return nil
	}
	desc, ok := obj.(*generator.Descriptor)
	if ok {
		return desc
	}
	return nil
}

func (m *mex) generateFields(names, nums []string, desc *generator.Descriptor) {
	message := desc.DescriptorProto
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		name := generator.CamelCase(*field.Name)
		num := fmt.Sprintf("%d", *field.Number)
		m.P("const ", strings.Join(append(names, name), ""), " = \"", strings.Join(append(nums, num), "."), "\"")
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.generateFields(append(names, name), append(nums, num), subDesc)
		}
	}
}

func (m *mex) markDiff(names []string, name string) {
	// set field and all parent fields
	names = append(names, name)
	for len(names) > 1 {
		fieldName := strings.Join(names, "")
		m.P("fields.Set(", fieldName, ")")
		names = names[:len(names)-1]
	}
}

// generator.EnumDescriptor as formal arg type ?
func (m *mex) generateIsKeyField(parents, names []string, desc *generator.Descriptor) {
	message := desc.DescriptorProto
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		name := generator.CamelCase(*field.Name)
		fieldKey := strings.Join(append(names, name), "")
		m.P("return strings.HasPrefix(s, ", fieldKey, "+\".\") || s == ", fieldKey)
		m.importStrings = true
		return
	}
}

func (m *mex) generateDiffFields(parents, names []string, desc *generator.Descriptor) {
	message := desc.DescriptorProto
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_GROUP {
			// deprecated in proto3
			continue
		}

		name := generator.CamelCase(*field.Name)
		hierName := strings.Join(append(parents, name), ".")
		idx := ""
		mapType := m.support.GetMapType(m.gen, field)
		loop := false
		skipMap := false
		nullableMessage := false
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE && gogoproto.IsNullable(field) {
			nullableMessage = true
		}
		if nullableMessage {
			m.P("if m.", hierName, " != nil && o.", hierName, " != nil {")
		}

		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED ||
			*field.Type == descriptor.FieldDescriptorProto_TYPE_BYTES {
			depth := fmt.Sprintf("%d", len(parents))
			m.P("if len(m.", hierName, ") != len(o.", hierName, ") {")
			m.markDiff(names, name)
			m.P("} else {")
			if mapType == nil {
				m.P("for i", depth, " := 0; i", depth, " < len(m.", hierName, "); i", depth, "++ {")
				idx = "[i" + depth + "]"
			} else {
				m.P("for k", depth, ", _ := range m.", hierName, " {")
				m.P("_, vok", depth, " := o.", hierName, "[k", depth, "]")
				m.P("if !vok", depth, " {")
				m.markDiff(names, name)
				m.P("} else {")
				if !mapType.ValIsMessage {
					skipMap = true
				}
				idx = "[k" + depth + "]"
			}
			loop = true
		}
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE && !skipMap {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			subNames := append(names, name)
			if mapType != nil {
				subDesc = gensupport.GetDesc(m.gen, mapType.ValField.GetTypeName())
				subNames = append(subNames, "Value")
			}
			m.generateDiffFields(append(parents, name+idx), subNames, subDesc)
		} else {
			m.P("if m.", hierName, idx, " != o.", hierName, idx, " {")
			m.markDiff(names, name)
			if loop {
				m.P("break")
			}
			m.P("}")
		}
		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED ||
			*field.Type == descriptor.FieldDescriptorProto_TYPE_BYTES {
			m.P("}")
			m.P("}")
			if mapType != nil {
				m.P("}")
			}
		}
		if nullableMessage {
			m.P("} else if (m.", hierName, " != nil && o.", hierName, " == nil) || (m.", hierName, " == nil && o.", hierName, " != nil) {")
			m.markDiff(names, name)
			m.P("}")
		}
	}
}

type AllFieldsGen int

const (
	AllFieldsGenSlice = iota
	AllFieldsGenMap
)

func (m *mex) generateAllFields(afg AllFieldsGen, names, nums []string, desc *generator.Descriptor) {
	message := desc.DescriptorProto
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		name := generator.CamelCase(*field.Name)
		num := fmt.Sprintf("%d", *field.Number)
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.generateAllFields(afg, append(names, name), append(nums, num), subDesc)
		default:
			switch afg {
			case AllFieldsGenSlice:
				m.P(strings.Join(append(names, name), ""), ",")
			case AllFieldsGenMap:
				m.P(strings.Join(append(names, name), ""), ": struct{}{},")
			}
		}
	}
}

func (m *mex) generateMethodFields(fieldPrefix string, names []string, noconfigMap map[string]struct{}, desc *generator.Descriptor, method *descriptor.MethodDescriptorProto) {
	message := desc.DescriptorProto
	noconfig := gensupport.GetNoConfig(message, method)
	for _, nc := range strings.Split(noconfig, ",") {
		name := strings.Replace(fieldPrefix+nc, ".", "", -1)
		noconfigMap[name] = struct{}{}
	}
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if keyField := gensupport.GetMessageKey(message); keyField != nil {
			// only skip key for the top-level message
			if *keyField.Name == *field.Name && len(names) == 1 {
				continue
			}
		}
		name := generator.CamelCase(*field.Name)
		fieldName := strings.Join(append(names, name), "")
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			if _, ok := noconfigMap[fieldName]; ok {
				continue
			}
			m.P(fieldName, ": struct{}{},")
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.generateMethodFields(fieldPrefix, append(names, name), noconfigMap, subDesc, method)
		default:
			if _, ok := noconfigMap[fieldName]; !ok {
				m.P(fieldName, ": struct{}{},")
			}
		}
	}
}

// Generate a simple string map to use in user-friendly error messages EC-608
func (m *mex) generateAllStringFieldsMap(afg AllFieldsGen, names, nums []string, fprefix string, desc *generator.Descriptor) {

	message := desc.DescriptorProto
	for ii, field := range message.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		name := generator.CamelCase(*field.Name)
		pname := name
		num := fmt.Sprintf("%d", *field.Number)

		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.generateAllStringFieldsMap(afg, append(names, name), append(nums, num), fprefix, subDesc)
		default:

			switch afg {

			case AllFieldsGenSlice:
				m.P(strings.Join(append(names, name), ""), ",")

			case AllFieldsGenMap:
				var readable []string
				pname = strings.Join(append(names, name, ""), "")
				m.P(fprefix, strings.Join(append(names, name), ""), ":")

				l := 0
				// take the camelcase name and insert " " before
				// each capital letter, use as the value of map
				//
				for s := pname; s != ""; s = s[l:] {
					l = strings.IndexFunc(s[1:], unicode.IsUpper) + 1
					if l <= 0 {
						l = len(s)
					}
					readable = append(readable, s[:l])
				}
				pstr := strings.Join(readable, " ") // readable?
				m.P("\"", pstr, "\"", ",")
				readable = nil
			}
		}
	}
}

func (m *mex) generateListAddRemoves(parents []string, top, desc *generator.Descriptor, visited []*generator.Descriptor) {
	if gensupport.WasVisited(desc, visited) {
		return
	}
	msgtyp := m.gen.TypeName(top)
	for ii, field := range desc.DescriptorProto.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		name := generator.CamelCase(*field.Name)
		if *field.Label != descriptor.FieldDescriptorProto_LABEL_REPEATED {
			if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
				m.generateListAddRemoves(append(parents, name), top, subDesc, append(visited, desc))
			}
			continue
		}
		mapType := m.support.GetMapType(m.gen, field)
		if mapType != nil {
			continue
		}
		hierName := strings.Join(append(parents, name), ".")
		hierFuncName := strings.Join(append(parents, name), "")
		ref := ""
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE && gogoproto.IsNullable(field) {
			ref = "*"
		}
		typ := ""
		keyValFunc := ".String()"
		keyType := "string"

		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			subMsg := subDesc.DescriptorProto
			typ = m.support.FQTypeName(m.gen, subDesc)
			if GetObjKey(subMsg) || gensupport.GetObjAndKey(subMsg) {
				keyValFunc = ".GetKeyString()"
			} else if gensupport.GetMessageKey(subMsg) != nil || gensupport.GetStringKeyField(subMsg) != "" {
				keyValFunc = ".GetKey().GetKeyString()"
			}
		} else {
			typ = m.support.GoType(m.gen, field)
			keyType = typ
			keyValFunc = ""
		}

		m.P("func (m *", msgtyp, ") Add", hierFuncName, "(vals... ", ref, typ, ") int {")
		m.P("changes := 0")
		m.P("cur := make(map[", keyType, "]struct{})")
		m.P("for _, v := range m.", hierName, "{")
		m.P("  cur[v", keyValFunc, "] = struct{}{}")
		m.P("}")
		m.P("for _, v := range vals {")
		m.P("  if _, found := cur[v", keyValFunc, "]; found {")
		m.P("    continue // duplicate")
		m.P("  }")
		m.P("  m.", hierName, "= append(m.", hierName, ", v)")
		m.P("  changes++")
		m.P("}")
		m.P("return changes")
		m.P("}")
		m.P()

		m.P("func (m *", msgtyp, ") Remove", hierFuncName, "(vals... ", ref, typ, ") int {")
		m.P("changes := 0")
		m.P("remove := make(map[", keyType, "]struct{})")
		m.P("for _, v := range vals {")
		m.P("  remove[v", keyValFunc, "] = struct{}{}")
		m.P("}")
		m.P("for i := len(m.", hierName, "); i >= 0; i-- {")
		m.P("  if _, found := remove[m.", hierName, "[i]", keyValFunc, "]; found {")
		m.P("    m.", hierName, " = append(m.", hierName, "[:i], m.", hierName, "[i+1:]...)")
		m.P("    changes++")
		m.P("  }")
		m.P("}")
		m.P("return changes")
		m.P("}")
		m.P()
	}
}

func (m *mex) needsUpdateListActionVar(desc *generator.Descriptor, visited []*generator.Descriptor) bool {
	if gensupport.WasVisited(desc, visited) {
		return false
	}
	for ii, field := range desc.DescriptorProto.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			return true
		}
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			if m.needsUpdateListActionVar(subDesc, append(visited, desc)) {
				return true
			}
		}
	}
	return false
}

func (m *mex) generateCopyIn(parents, nums []string, desc *generator.Descriptor, visited []*generator.Descriptor, hasGrpcFields bool) {
	if gensupport.WasVisited(desc, visited) {
		return
	}
	for ii, field := range desc.DescriptorProto.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if field.OneofIndex != nil {
			// no support for copy OneOf fields
			continue
		}
		if *field.Name == gensupport.UpdateListActionField {
			continue
		}

		name := generator.CamelCase(*field.Name)
		hierName := strings.Join(append(parents, name), ".")
		hierFuncName := strings.Join(append(parents, name), "")
		num := fmt.Sprintf("%d", *field.Number)
		idx := ""
		nullableMessage := false
		ref := ""
		deref := "*"
		isMessage := *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE
		if isMessage && gogoproto.IsNullable(field) {
			nullableMessage = true
			ref = "*"
			deref = ""
		}
		mapType := m.support.GetMapType(m.gen, field)
		skipMessage := false

		numStr := strings.Join(append(nums, num), ".")
		if hasGrpcFields && isMessage {
			m.P("if fmap.HasOrHasChild(\"", numStr, "\") {")
		} else if hasGrpcFields {
			m.P("if fmap.Has(\"", numStr, "\") {")
		}
		if nullableMessage || *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			m.P("if src.", hierName, " != nil {")
		}
		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			depth := fmt.Sprintf("%d", len(parents))
			idx = "[k" + depth + "]"
			m.P("if updateListAction == \"", util.UpdateListActionAdd, "\" {")
			// add
			if mapType == nil {
				// punt to user to add to list, in case they want to avoid duplicates
				m.P("changed += m.Add", hierFuncName, "(src.", hierName, "...)")
			} else {
				m.P("for k", depth, ", v := range src.", hierName, " {")
				if mapType.ValIsMessage {
					m.P("v = ", deref, "v.Clone()")
				}
				m.P("m.", hierName, idx, " = v")
				m.P("changed++")
				m.P("}")
			}
			m.P("} else if updateListAction == \"", util.UpdateListActionRemove, "\" {")
			// remove
			if mapType == nil {
				// punt to user to remove from list
				m.P("changed += m.Remove", hierFuncName, "(src.", hierName, "...)")
			} else {
				m.P("for k", depth, ", _ := range src.", hierName, " {")
				m.P("if _, ok := m.", hierName, idx, "; ok {")
				m.P("delete(m.", hierName, ", k", depth, ")")
				m.P("changed++")
				m.P("}")
				m.P("}")
			}
			m.P("} else {")
			// replace
			m.printCopyInMakeArray(hierName, desc, field)
			if mapType == nil {
				if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
					typ := m.support.FQTypeName(m.gen, subDesc)
					m.P("m.", hierName, " = make([]", ref, typ, ", 0)")
					m.P("for k", depth, ", _ := range src.", hierName, " {")
					m.P("m.", hierName, " = append(m.", hierName, ", ", deref, "src.", hierName, idx, ".Clone())")
					m.P("}")
				} else {
					m.P("m.", hierName, " = make([]", ref, m.support.GoType(m.gen, field), ", 0)")
					m.P("m.", hierName, " = append(m.", hierName, ", src.", hierName, "...)")
				}
			} else {
				m.P("for k", depth, ", v := range src.", hierName, " {")
				if mapType.ValIsMessage {
					m.P("m.", hierName, idx, " = ", deref, "v.Clone()")
				} else {
					m.P("m.", hierName, idx, " = v")
				}
				m.P("}")
			}
			m.P("changed++")
			m.P("}")
		} else {
			switch *field.Type {
			case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
				if skipMessage {
					break
				}
				subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
				if mapType != nil {
					if mapType.ValIsMessage {
						m.P("m.", hierName, idx, " = &", mapType.ValType, "{}")
					}
					subDesc = gensupport.GetDesc(m.gen, mapType.ValField.GetTypeName())
				} else if gogoproto.IsNullable(field) {
					typ := m.support.FQTypeName(m.gen, subDesc)
					m.P("if m.", hierName, idx, " == nil {")
					m.P("m.", hierName, idx, " = &", typ, "{}")
					m.P("}")
				}
				subHasGrpcFields := hasGrpcFields
				if GetCopyInAllFields(subDesc.DescriptorProto) {
					// copy in all subdata, do so by ignoring fields checks
					subHasGrpcFields = false
				}
				m.generateCopyIn(append(parents, name+idx), append(nums, num), subDesc, append(visited, desc), subHasGrpcFields)
			case descriptor.FieldDescriptorProto_TYPE_GROUP:
				// deprecated in proto3
			case descriptor.FieldDescriptorProto_TYPE_BYTES:
				m.printCopyInMakeArray(hierName, desc, field)
				m.P("if src.", hierName, " != nil {")
				m.P("m.", hierName, " = src.", hierName)
				m.P("changed++")
				m.P("}")
			default:
				if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
					m.P("m.", hierName, " = src.", hierName)
					m.P("changed++")
				} else {
					m.P("if m.", hierName, " != src.", hierName, "{")
					m.P("m.", hierName, " = src.", hierName)
					m.P("changed++")
					m.P("}")
				}
			}
		}
		if nullableMessage || *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			m.P("} else if m.", hierName, " != nil {")
			m.P("m.", hierName, " = nil")
			m.P("changed++")
			m.P("}")
		}
		if hasGrpcFields {
			m.P("}")
		}
	}
}

func (m *mex) generateDeepCopyIn(desc *generator.Descriptor) {
	msgtyp := m.gen.TypeName(desc)
	m.P("func (m *", msgtyp, ") DeepCopyIn(src *", msgtyp, ") {")
	for ii, field := range desc.DescriptorProto.Field {
		if ii == 0 && *field.Name == "fields" {
			continue
		}
		if field.OneofIndex != nil {
			// no support
			continue
		}
		name := generator.CamelCase(*field.Name)
		nullable := false
		checkField := field
		mapType := m.support.GetMapType(m.gen, field)
		if mapType != nil {
			checkField = mapType.ValField
		}
		if *checkField.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			nullable = gogoproto.IsNullable(field)
		}
		ptr := ""
		if nullable {
			ptr = "*"
		}
		nilCheck := false
		if nullable || *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			m.P("if src.", name, " != nil {")
			nilCheck = true
		}
		if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			ftype := m.support.GoType(m.gen, field)
			if mapType == nil {
				m.P("m.", name, " = make([]", ptr, ftype, ", len(src.", name, "), len(src.", name, "))")
				m.P("for ii, s := range src.", name, " {")
				to := "m." + name + "[ii]"
				m.printCopyVar(field, to, "s", nullable, mapType)
			} else {
				m.P("m.", name, " = make(map[", mapType.KeyType, "]", ptr, mapType.ValType, ")")
				m.P("for k, v := range src.", name, " {")
				to := "m." + name + "[k]"
				m.printCopyVar(mapType.ValField, to, "v", nullable, mapType)
			}
			m.P("}")
		} else {
			m.printCopyVar(field, "m."+name, "src."+name, nullable, mapType)
		}
		if nilCheck {
			m.P("} else {")
			m.P("m.", name, " = nil")
			m.P("}")
		}
	}
	m.P("}")
	m.P()
}

func (m *mex) printCopyVar(field *descriptor.FieldDescriptorProto, to, from string, nullable bool, mapType *gensupport.MapType) {
	deepCopy := false
	if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		desc := gensupport.GetDesc(m.gen, field.GetTypeName())
		// check if this object is from our proto files
		// (as opposed to something like google_protobuf.Timestamp)
		if m.support.GenFile(*desc.File().Name) {
			deepCopy = true
		}
	}
	tmp := to
	deepCopyRef := "&"
	copyDeref := ""
	useTempVar := nullable
	if mapType != nil && mapType.ValIsMessage && deepCopy {
		useTempVar = true
	}
	if useTempVar {
		// use temp var
		tmp = strings.TrimPrefix(from, "src.")
		tmp = "tmp_" + tmp
		ftype := m.support.GoType(m.gen, field)
		m.P("var ", tmp, " ", ftype)
		if nullable {
			deepCopyRef = ""
			copyDeref = "*"
		}
	}
	if deepCopy {
		m.P(tmp, ".DeepCopyIn(", deepCopyRef, from, ")")
	} else {
		m.P(tmp, " = ", copyDeref, from)
	}
	if useTempVar {
		ref := ""
		if nullable {
			ref = "&"
		}
		m.P(to, " = ", ref, tmp)
	}
}

type cudTemplateArgs struct {
	Name        string
	KeyType     string
	CudName     string
	HasFields   bool
	GenCache    bool
	NotifyCache bool
	ObjAndKey   bool
	RefBys      []string
}

var fieldsValTemplate = `
func (m *{{.Name}}) ValidateUpdateFields() error {
	return m.ValidateUpdateFieldsCustom(Update{{.Name}}FieldsMap)
}

func (m *{{.Name}}) ValidateUpdateFieldsCustom(allowedFields *FieldMap) error {
	if m.Fields == nil {
		return fmt.Errorf("nothing specified to update")
	}
	fmap := MakeFieldMap(m.Fields)
	badFieldStrs := []string{}
	for _, field := range fmap.Fields() {
		if m.IsKeyField(field) {
			continue
		}
		if !allowedFields.Has(field) {
			if _, ok := {{.Name}}AllFieldsStringMap[field]; !ok {
				continue
			}
			badFieldStrs = append(badFieldStrs, {{.Name}}AllFieldsStringMap[field])
		}
	}
	if len(badFieldStrs) > 0 {
		return fmt.Errorf("specified field(s) %s cannot be modified", strings.Join(badFieldStrs, ","))
	}
	return nil
}

`

var cudTemplateIn = `
func (s *{{.Name}}) HasFields() bool {
{{- if (.HasFields)}}
	return true
{{- else}}
	return false
{{- end}}
}

type {{.Name}}Store interface {
	Create(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error)
	Update(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error)
	Delete(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error)
	Put(ctx context.Context, m *{{.Name}}, wait func(int64), ops ...objstore.KVOp) (*Result, error)
	LoadOne(key string) (*{{.Name}}, int64, error)
	Get(ctx context.Context, key *{{.KeyType}}, buf *{{.Name}}) bool
	STMGet(stm concurrency.STM, key *{{.KeyType}}, buf *{{.Name}}) bool
	STMPut(stm concurrency.STM, obj *{{.Name}}, ops ...objstore.KVOp)
	STMDel(stm concurrency.STM, key *{{.KeyType}})
	STMHas(stm concurrency.STM, key *{{.KeyType}}) bool
}

type {{.Name}}StoreImpl struct {
	kvstore objstore.KVStore
}

func New{{.Name}}Store(kvstore objstore.KVStore) *{{.Name}}StoreImpl {
	return &{{.Name}}StoreImpl{kvstore: kvstore}
}

func (s *{{.Name}}StoreImpl) Create(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error) {
{{- if (.ObjAndKey)}}
	err := m.ValidateKey()
{{- else if (.HasFields)}}
	err := m.Validate({{.Name}}AllFieldsMap)
{{- else}}
	err := m.Validate(nil)
{{- end}}
	if err != nil { return nil, err }
	key := objstore.DbKeyString("{{.Name}}", m.GetKey())
	val, err := json.Marshal(m)
	if err != nil { return nil, err }
	rev, err := s.kvstore.Create(ctx, key, string(val))
	if err != nil { return nil, err }
	if wait != nil {
		wait(rev)
	}
	return &Result{}, err
}

func (s *{{.Name}}StoreImpl) Update(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error) {
{{- if (.ObjAndKey)}}
	err := m.ValidateKey()
{{- else if (.HasFields)}}
	fmap := MakeFieldMap(m.Fields)
	err := m.Validate(fmap)
{{- else}}
	err := m.Validate(nil)
{{- end}}
	if err != nil { return nil, err }
	key := objstore.DbKeyString("{{.Name}}", m.GetKey())
	var vers int64 = 0
{{- if (.HasFields)}}
	curBytes, vers, _, err := s.kvstore.Get(key)
	if err != nil { return nil, err }
	var cur {{.Name}}
	err = json.Unmarshal(curBytes, &cur)
	if err != nil { return nil, err }
	cur.CopyInFields(m)
	// never save fields
	cur.Fields = nil
	val, err := json.Marshal(cur)
{{- else}}
	val, err := json.Marshal(m)
{{- end}}
	if err != nil { return nil, err }
	rev, err := s.kvstore.Update(ctx, key, string(val), vers)
	if err != nil { return nil, err }
	if wait != nil {
		wait(rev)
	}
	return &Result{}, err
}

func (s *{{.Name}}StoreImpl) Put(ctx context.Context, m *{{.Name}}, wait func(int64), ops ...objstore.KVOp) (*Result, error) {
{{- if (.ObjAndKey)}}
	err := m.ValidateKey()
{{- else if (.HasFields)}}
	err := m.Validate({{.Name}}AllFieldsMap)
	m.Fields = nil
{{- else}}
	err := m.Validate(nil)
{{- end}}
	if err != nil { return nil, err }
	key := objstore.DbKeyString("{{.Name}}", m.GetKey())
	var val []byte
	val, err = json.Marshal(m)
	if err != nil { return nil, err }
	rev, err := s.kvstore.Put(ctx, key, string(val), ops...)
	if err != nil { return nil, err }
	if wait != nil {
		wait(rev)
	}
	return &Result{}, err
}

func (s *{{.Name}}StoreImpl) Delete(ctx context.Context, m *{{.Name}}, wait func(int64)) (*Result, error) {
	err := m.GetKey().ValidateKey()
	if err != nil { return nil, err }
	key := objstore.DbKeyString("{{.Name}}", m.GetKey())
	rev, err := s.kvstore.Delete(ctx, key)
	if err != nil { return nil, err }
	if wait != nil {
		wait(rev)
	}
	return &Result{}, err
}

func (s *{{.Name}}StoreImpl) LoadOne(key string) (*{{.Name}}, int64, error) {
	val, rev, _, err := s.kvstore.Get(key)
	if err != nil {
		return nil, 0, err
	}
	var obj {{.Name}}
	err = json.Unmarshal(val, &obj)
	if err != nil {
		log.DebugLog(log.DebugLevelApi, "Failed to parse {{.Name}} data", "val", string(val), "err", err)
		return nil, 0, err
	}
	return &obj, rev, nil
}

func (s *{{.Name}}StoreImpl) Get(ctx context.Context, key *{{.KeyType}}, buf *{{.Name}}) bool {
	keystr := objstore.DbKeyString("{{.Name}}", key)
	val, _, _, err := s.kvstore.Get(keystr)
	if err != nil {
		return false
	}
	return s.parseGetData(val, buf)
}

func (s *{{.Name}}StoreImpl) STMGet(stm concurrency.STM, key *{{.KeyType}}, buf *{{.Name}}) bool {
	keystr := objstore.DbKeyString("{{.Name}}", key)
	valstr := stm.Get(keystr)
	return s.parseGetData([]byte(valstr), buf)
}

func (s *{{.Name}}StoreImpl) STMHas(stm concurrency.STM, key *{{.KeyType}}) bool {
	keystr := objstore.DbKeyString("{{.Name}}", key)
	return stm.Get(keystr) != ""
}

func (s *{{.Name}}StoreImpl) parseGetData(val []byte, buf *{{.Name}}) bool {
	if len(val) == 0 {
		return false
	}
	if buf != nil {
		// clear buf, because empty values in val won't
		// overwrite non-empty values in buf.
		*buf = {{.Name}}{}
		err := json.Unmarshal(val, buf)
		if err != nil {
			return false
		}
	}
	return true
}

func (s *{{.Name}}StoreImpl) STMPut(stm concurrency.STM, obj *{{.Name}}, ops ...objstore.KVOp) {
	keystr := objstore.DbKeyString("{{.Name}}", obj.GetKey())

	val, err := json.Marshal(obj)
	if err != nil {
		log.InfoLog("{{.Name}} json marshal failed", "obj", obj, "err", err)
	}
	v3opts := GetSTMOpts(ops...)
	stm.Put(keystr, string(val), v3opts...)
}

func (s *{{.Name}}StoreImpl) STMDel(stm concurrency.STM, key *{{.KeyType}}) {
	keystr := objstore.DbKeyString("{{.Name}}", key)
	stm.Del(keystr)
}

func StoreList{{.Name}}(ctx context.Context, kvstore objstore.KVStore) ([]{{.Name}}, error) {
	keyPrefix := objstore.DbKeyPrefixString("{{.Name}}")+"/"
	objs := []{{.Name}}{}
	err := kvstore.List(keyPrefix, func(key, val []byte, rev, modRev int64) error {
		obj := {{.Name}}{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return fmt.Errorf("failed to unmarshal {{.Name}} json %s, %s", string(val), err)
		}
		objs = append(objs, obj)
		return nil
	})
	return objs, err
}

`

type cacheTemplateArgs struct {
	Name           string
	KeyType        string
	CudCache       bool
	NotifyCache    bool
	NotifyFlush    bool
	ParentObjName  string
	WaitForState   string
	ObjAndKey      bool
	CustomKeyType  string
	StreamOut      bool
	StringKeyField string
	StateFieldType string
}

var cacheTemplateIn = `
type {{.Name}}KeyWatcher struct {
	cb func(ctx context.Context)
}

type {{.Name}}CacheData struct {
	Obj *{{.Name}}
	ModRev int64
}

func (s *{{.Name}}CacheData) Clone() *{{.Name}}CacheData {
	cp := {{.Name}}CacheData{}
	if s.Obj != nil {
		cp.Obj = &{{.Name}}{}
		cp.Obj.DeepCopyIn(s.Obj)
	}
	cp.ModRev = s.ModRev
	return &cp
}

// {{.Name}}Cache caches {{.Name}} objects in memory in a hash table
// and keeps them in sync with the database.
type {{.Name}}Cache struct {
	Objs map[{{.KeyType}}]*{{.Name}}CacheData
	Mux util.Mutex
	List map[{{.KeyType}}]struct{}
	FlushAll bool
	NotifyCbs []func(ctx context.Context, obj *{{.Name}}, modRev int64)
	UpdatedCbs []func(ctx context.Context, old *{{.Name}}, new *{{.Name}})
	DeletedCbs []func(ctx context.Context, old *{{.Name}})
	KeyWatchers map[{{.KeyType}}][]*{{.Name}}KeyWatcher
	UpdatedKeyCbs []func(ctx context.Context, key *{{.KeyType}})
	DeletedKeyCbs []func(ctx context.Context, key *{{.KeyType}})
{{- if .CudCache}}
	Store {{.Name}}Store
{{- end}}
}

func New{{.Name}}Cache() *{{.Name}}Cache {
	cache := {{.Name}}Cache{}
	Init{{.Name}}Cache(&cache)
	return &cache
}

func Init{{.Name}}Cache(cache *{{.Name}}Cache) {
	cache.Objs = make(map[{{.KeyType}}]*{{.Name}}CacheData)
	cache.KeyWatchers = make(map[{{.KeyType}}][]*{{.Name}}KeyWatcher)
	cache.NotifyCbs = nil
	cache.UpdatedCbs = nil
	cache.DeletedCbs = nil
	cache.UpdatedKeyCbs = nil
	cache.DeletedKeyCbs = nil
}

func (c *{{.Name}}Cache) GetTypeString() string {
	return "{{.Name}}"
}

func (c *{{.Name}}Cache) Get(key *{{.KeyType}}, valbuf *{{.Name}}) bool {
	var modRev int64
	return c.GetWithRev(key, valbuf, &modRev)
}

{{- if .CudCache}}
// STMGet gets from the store if STM is set, otherwise gets from cache
func (c *{{.Name}}Cache) STMGet(ostm *OptionalSTM, key *{{.KeyType}}, valbuf *{{.Name}}) bool {
	if ostm.stm != nil {
		if c.Store == nil {
			// panic, otherwise if we fallback to cache, we may silently
			// introduce race conditions and intermittent failures due to
			// reading from cache during a transaction.
			panic("{{.Name}}Cache store not set, cannot read via STM")
		}
		return c.Store.STMGet(ostm.stm, key, valbuf)
	}
	var modRev int64
	return c.GetWithRev(key, valbuf, &modRev)
}
{{- end}}

func (c *{{.Name}}Cache) GetWithRev(key *{{.KeyType}}, valbuf *{{.Name}}, modRev *int64) bool {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	inst, found := c.Objs[*key]
	if found {
		valbuf.DeepCopyIn(inst.Obj)
		*modRev = inst.ModRev
	}
	return found
}

func (c *{{.Name}}Cache) HasKey(key *{{.KeyType}}) bool {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	_, found := c.Objs[*key]
	return found
}

func (c *{{.Name}}Cache) GetAllKeys(ctx context.Context, cb func(key *{{.KeyType}}, modRev int64)) {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	for key, data := range c.Objs {
		cb(&key, data.ModRev)
	}
}

func (c *{{.Name}}Cache) GetAllLocked(ctx context.Context, cb func(obj *{{.Name}}, modRev int64)) {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	for _, data := range c.Objs {
		cb(data.Obj, data.ModRev)
	}
}

func (c *{{.Name}}Cache) Update(ctx context.Context, in *{{.Name}}, modRev int64) {
	c.UpdateModFunc(ctx, in.GetKey(), modRev, func(old *{{.Name}}) (*{{.Name}}, bool) {
		return in, true
	})
}

func (c *{{.Name}}Cache) UpdateModFunc(ctx context.Context, key *{{.KeyType}}, modRev int64, modFunc func(old *{{.Name}}) (new *{{.Name}}, changed bool)) {
	c.Mux.Lock()
	var old *{{.Name}}
	if oldData, found := c.Objs[*key]; found {
		old = oldData.Obj
	}
	new, changed := modFunc(old)
	if !changed {
		c.Mux.Unlock()
		return
	}
	if len(c.UpdatedCbs) > 0 || len(c.NotifyCbs) > 0 {
		newCopy := &{{.Name}}{}
		newCopy.DeepCopyIn(new)
		for _, cb := range c.UpdatedCbs {
			defer cb(ctx, old, newCopy)
		}
		for _, cb := range c.NotifyCbs {
			if cb != nil {
				defer cb(ctx, newCopy, modRev)
			}
		}
	}
	for _, cb := range c.UpdatedKeyCbs {
		defer cb(ctx, key)
	}
	store := &{{.Name}}{}
	store.DeepCopyIn(new)
	c.Objs[new.GetKeyVal()] = &{{.Name}}CacheData{
		Obj: store,
		ModRev: modRev,
	}
	log.SpanLog(ctx, log.DebugLevelApi, "cache update", "new", store)
	c.Mux.Unlock()
	c.TriggerKeyWatchers(ctx, new.GetKey())
}

func (c *{{.Name}}Cache) Delete(ctx context.Context, in *{{.Name}}, modRev int64) {
	c.DeleteCondFunc(ctx, in, modRev, func(old *{{.Name}}) bool {
		return true
	})
}

func (c *{{.Name}}Cache) DeleteCondFunc(ctx context.Context, in *{{.Name}}, modRev int64, condFunc func(old *{{.Name}}) bool) {
	c.Mux.Lock()
	var old *{{.Name}}
	oldData, found := c.Objs[in.GetKeyVal()]
	if found {
		old = oldData.Obj
		if !condFunc(old) {
			c.Mux.Unlock()
			return
		}
	}
	delete(c.Objs, in.GetKeyVal())
	log.SpanLog(ctx, log.DebugLevelApi, "cache delete", "key", in.GetKeyVal())
	c.Mux.Unlock()
	obj := old
	if obj == nil {
		obj = in
	}
	for _, cb := range c.NotifyCbs {
		if cb != nil {
			cb(ctx, obj, modRev)
		}
	}
	if old != nil {
		for _, cb := range c.DeletedCbs {
			cb(ctx, old)
		}
	}
	for _, cb := range c.DeletedKeyCbs {
		cb(ctx, in.GetKey())
	}
	c.TriggerKeyWatchers(ctx, in.GetKey())
}

func (c *{{.Name}}Cache) Prune(ctx context.Context, validKeys map[{{.KeyType}}]struct{}) {
	log.SpanLog(ctx, log.DebugLevelApi, "Prune {{.Name}}", "numValidKeys", len(validKeys))
	notify := make(map[{{.KeyType}}]*{{.Name}}CacheData)
	c.Mux.Lock()
	for key, _ := range c.Objs {
		if _, ok := validKeys[key]; !ok {
			if len(c.NotifyCbs) > 0 || len(c.DeletedKeyCbs) > 0 || len(c.DeletedCbs) > 0 {
				notify[key] = c.Objs[key]
			}
			delete(c.Objs, key)
		}
	}
	c.Mux.Unlock()
	for key, old := range notify {
		obj := old.Obj
		if obj == nil {
			obj = &{{.Name}}{}
			obj.SetKey(&key)
		}
	        for _, cb := range c.NotifyCbs {
			if cb != nil {
				cb(ctx, obj, old.ModRev)
			}
		}
		for _, cb := range c.DeletedKeyCbs {
			cb(ctx, &key)
		}
		if old.Obj != nil {
			for _, cb := range c.DeletedCbs {
				cb(ctx, old.Obj)
			}
		}
		c.TriggerKeyWatchers(ctx, &key)
	}
}

func (c *{{.Name}}Cache) GetCount() int {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	return len(c.Objs)
}

func (c *{{.Name}}Cache) Flush(ctx context.Context, notifyId int64) {
{{- if .NotifyFlush}}
	log.SpanLog(ctx, log.DebugLevelApi, "CacheFlush {{.Name}}", "notifyId", notifyId, "FlushAll", c.FlushAll)
	flushed := make(map[{{.KeyType}}]*{{.Name}}CacheData)
	c.Mux.Lock()
	for key, val := range c.Objs {
		if !c.FlushAll && val.Obj.NotifyId != notifyId {
			continue
		}
		flushed[key] = c.Objs[key]
		log.SpanLog(ctx, log.DebugLevelApi, "CacheFlush {{.Name}} delete", "key", key)
		delete(c.Objs, key)
	}
	c.Mux.Unlock()
	if len(flushed) > 0 {
		for key, old := range flushed {
			obj := old.Obj
			if obj == nil {
				obj = &{{.Name}}{}
				obj.SetKey(&key)
			}
		        for _, cb := range c.NotifyCbs {
				if cb != nil {
					cb(ctx, obj, old.ModRev)
				}
			}
			for _, cb := range c.DeletedKeyCbs {
				cb(ctx, &key)
			}
			if old.Obj != nil {
				for _, cb := range c.DeletedCbs {
					cb(ctx, old.Obj)
				}
			}
			c.TriggerKeyWatchers(ctx, &key)
		}
	}
{{- end}}
}

func (c *{{.Name}}Cache) Show(filter *{{.Name}}, cb func(ret *{{.Name}}) error) error {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	for _, data := range c.Objs {
{{- if .CudCache}}
		if !data.Obj.Matches(filter, MatchFilter()) {
			continue
		}
{{- end}}
		err := cb(data.Obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func {{.Name}}GenericNotifyCb(fn func(key *{{.KeyType}}, old *{{.Name}})) func(objstore.ObjKey, objstore.Obj) {
	return func(objkey objstore.ObjKey, obj objstore.Obj) {
		fn(objkey.(*{{.KeyType}}), obj.(*{{.Name}}))
	}
}


func (c *{{.Name}}Cache) SetNotifyCb(fn func(ctx context.Context, obj *{{.Name}}, modRev int64)) {
	c.NotifyCbs = []func(ctx context.Context, obj *{{.Name}}, modRev int64){fn}
}

func (c *{{.Name}}Cache) SetUpdatedCb(fn func(ctx context.Context, old *{{.Name}}, new *{{.Name}})) {
	c.UpdatedCbs = []func(ctx context.Context, old *{{.Name}}, new *{{.Name}}){fn}
}

func (c *{{.Name}}Cache) SetDeletedCb(fn func(ctx context.Context, old *{{.Name}})) {
	c.DeletedCbs = []func(ctx context.Context, old *{{.Name}}){fn}
}

func (c *{{.Name}}Cache) SetUpdatedKeyCb(fn func(ctx context.Context, key *{{.KeyType}})) {
	c.UpdatedKeyCbs = []func(ctx context.Context, key *{{.KeyType}}){fn}
}

func (c *{{.Name}}Cache) SetDeletedKeyCb(fn func(ctx context.Context, key *{{.KeyType}})) {
	c.DeletedKeyCbs = []func(ctx context.Context, key *{{.KeyType}}){fn}
}

func (c *{{.Name}}Cache) AddUpdatedCb(fn func(ctx context.Context, old *{{.Name}}, new *{{.Name}})) {
	c.UpdatedCbs = append(c.UpdatedCbs, fn)
}

func (c *{{.Name}}Cache) AddDeletedCb(fn func(ctx context.Context, old *{{.Name}})) {
	c.DeletedCbs = append(c.DeletedCbs, fn)
}

func (c *{{.Name}}Cache) AddNotifyCb(fn func(ctx context.Context, obj *{{.Name}}, modRev int64)) {
	c.NotifyCbs = append(c.NotifyCbs, fn)
}

func (c *{{.Name}}Cache) AddUpdatedKeyCb(fn func(ctx context.Context, key *{{.KeyType}})) {
	c.UpdatedKeyCbs = append(c.UpdatedKeyCbs, fn)
}

func (c *{{.Name}}Cache) AddDeletedKeyCb(fn func(ctx context.Context, key *{{.KeyType}})) {
	c.DeletedKeyCbs = append(c.DeletedKeyCbs, fn)
}

func (c *{{.Name}}Cache) SetFlushAll() {
	c.FlushAll = true
}

func (c *{{.Name}}Cache) WatchKey(key *{{.KeyType}}, cb func(ctx context.Context)) context.CancelFunc {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	list, ok := c.KeyWatchers[*key]
	if !ok {
		list = make([]*{{.Name}}KeyWatcher, 0)
	}
	watcher := {{.Name}}KeyWatcher{cb: cb}
	c.KeyWatchers[*key] = append(list, &watcher)
	log.DebugLog(log.DebugLevelApi, "Watching {{.Name}}", "key", key)
	return func() {
		c.Mux.Lock()
		defer c.Mux.Unlock()
		list, ok := c.KeyWatchers[*key]
		if !ok { return }
		for ii, _ := range list {
			if list[ii] != &watcher {
				continue
			}
			if len(list) == 1 {
				delete(c.KeyWatchers, *key)
				return
			}
			list[ii] = list[len(list)-1]
			list[len(list)-1] = nil
			c.KeyWatchers[*key] = list[:len(list)-1]
			return
		}
	}
}

func (c *{{.Name}}Cache) TriggerKeyWatchers(ctx context.Context, key *{{.KeyType}}) {
	watchers := make([]*{{.Name}}KeyWatcher, 0)
	c.Mux.Lock()
	if list, ok := c.KeyWatchers[*key]; ok {
		watchers = append(watchers, list...)
	}
	c.Mux.Unlock()
	for ii, _ := range watchers {
		watchers[ii].cb(ctx)
	}
}


{{- if .CudCache}}
// Note that we explicitly ignore the global revision number, because of the way
// the notify framework sends updates (by hashing keys and doing lookups, instead
// of sequentially through a history buffer), updates may be done out-of-order
// or multiple updates compressed into one update, so the state of the cache at
// any point in time may not by in sync with a particular database revision number.

func (c *{{.Name}}Cache) SyncUpdate(ctx context.Context, key, val []byte, rev, modRev int64) {
	obj := {{.Name}}{}
	err := json.Unmarshal(val, &obj)
	if err != nil {
		log.WarnLog("Failed to parse {{.Name}} data", "val", string(val), "err", err)
		return
	}
	c.Update(ctx, &obj, modRev)
	c.Mux.Lock()
	if c.List != nil {
		c.List[obj.GetKeyVal()] = struct{}{}
	}
	c.Mux.Unlock()
}

func (c *{{.Name}}Cache) SyncDelete(ctx context.Context, key []byte, rev, modRev int64) {
	obj := {{.Name}}{}
	keystr := objstore.DbKeyPrefixRemove(string(key))
{{- if .CustomKeyType}}
	{{.KeyType}}StringParse(keystr, &obj)
{{- else if (.StringKeyField) }}
    obj.{{.StringKeyField}} = keystr
{{- else}}
	{{.KeyType}}StringParse(keystr, obj.GetKey())
{{- end}}
	c.Delete(ctx, &obj, modRev)
}

func (c *{{.Name}}Cache) SyncListStart(ctx context.Context) {
	c.List = make(map[{{.KeyType}}]struct{})
}

func (c *{{.Name}}Cache) SyncListEnd(ctx context.Context) {
	deleted := make(map[{{.KeyType}}]*{{.Name}}CacheData)
	c.Mux.Lock()
	for key, val := range c.Objs {
		if _, found := c.List[key]; !found {
			deleted[key] = val
			delete(c.Objs, key)
		}
	}
	c.List = nil
	c.Mux.Unlock()
	for key, val := range deleted {
		obj := val.Obj
		if obj == nil {
			obj = &{{.Name}}{}
			obj.SetKey(&key)
		}
	        for _, cb := range c.NotifyCbs {
			if cb != nil {
				cb(ctx, obj, val.ModRev)
			}
		}
		for _, cb := range c.DeletedKeyCbs {
			cb(ctx, &key)
		}
		if val.Obj != nil {
			for _, cb := range c.DeletedCbs {
				cb(ctx, val.Obj)
			}
		}
		c.TriggerKeyWatchers(ctx, &key)
	}
}

func (s *{{.Name}}Cache) InitCacheWithSync(sync DataSync) {
	Init{{.Name}}Cache(s)
	s.InitSync(sync)
}

func (s *{{.Name}}Cache) InitSync(sync DataSync) {
	if sync != nil {
		s.Store = New{{.Name}}Store(sync.GetKVStore())
		sync.RegisterCache(s)
	}
}

func Init{{.Name}}CacheWithStore(cache *{{.Name}}Cache, store {{.Name}}Store) {
	Init{{.Name}}Cache(cache)
	cache.Store = store
}

{{- end}}

{{- if .ParentObjName}}
// {{.Name}}ObjectUpdater defines a way of updating a specific {{.Name}}
type {{.Name}}ObjectUpdater interface {
	// Get the current {{.Name}}
	Get() *{{.Name}}
	// Update the {{.Name}} for the specified Fields flags.
	Update(*{{.Name}}) error
}

// {{.Name}}Sender allows for streaming updates to {{.Name}}
type {{.Name}}Sender interface {
	// SendUpdate sends the updated object, fields without field flags set will be ignored
	SendUpdate(updateFn func(update *{{.Name}}) error) error
	// SendState sends an updated state. It will clear any errors unless
	// the WithStateError option is specified.
	SendState(state {{.StateFieldType}}, ops ...SenderOp) error
	// SendStatus appends the status message and sends it.
	SendStatus(updateType CacheUpdateType, message string, ops ...SenderOp) error
	// SendStatusIgnoreErr is the same as SendStatus but without error return
	// and without options to be compatible with older code.
	SendStatusIgnoreErr(updateType CacheUpdateType, message string)
}

// {{.Name}}SenderHelper implements {{.Name}}Sender
type {{.Name}}SenderHelper struct {
	updater {{.Name}}ObjectUpdater
}

func (s *{{.Name}}SenderHelper) SetUpdater(updater {{.Name}}ObjectUpdater) {
	s.updater = updater
}

// SendUpdate sends only the updated fields set by the Fields flags.
func (s *{{.Name}}SenderHelper) SendUpdate(updateFn func(update *{{.Name}}) error) error {
	obj := s.updater.Get()
	if err := updateFn(obj); err != nil {
		return err
	}
	return s.updater.Update(obj)
}

// SendState sends an updated state
func (s *{{.Name}}SenderHelper) SendState(state {{.StateFieldType}}, ops ...SenderOp) error {
	opts := GetSenderOptions(ops...)
	obj := s.updater.Get()
	obj.Fields = []string{
		{{.Name}}FieldState,
		{{.Name}}FieldErrors,
		{{.Name}}FieldStatus,
	}
	s.applyOpts(obj, opts)

	if opts.stateErr != nil {
		obj.Errors = []string{opts.stateErr.Error()}
	}
	obj.State = state
	obj.Status.SetTask({{.StateFieldType}}_CamelName[int32(state)])
	return s.updater.Update(obj)
}

// SendStatus appends the status message and sends it.
func (s *{{.Name}}SenderHelper) SendStatus(updateType CacheUpdateType, message string, ops ...SenderOp) error {
	opts := GetSenderOptions(ops...)
	obj := s.updater.Get()
	obj.Fields = []string{
		{{.Name}}FieldStatus,
	}
	s.applyOpts(obj, opts)

	switch updateType {
	case UpdateTask:
		obj.Status.SetTask(message)
	case UpdateStep:
		obj.Status.SetStep(message)
	}
	return s.updater.Update(obj)
}

func (s *{{.Name}}SenderHelper) SendStatusIgnoreErr(updateType CacheUpdateType, message string) {
	s.SendStatus(updateType, message)
}

func (s *{{.Name}}SenderHelper) applyOpts(obj *{{.Name}}, opts *SenderOptions) {
	if opts.resetStatus {
		obj.Fields = append(obj.Fields, {{.Name}}FieldStatus)
		obj.Status.StatusReset()
	}
}

// {{.Name}}CacheUpdater implements {{.Name}}Sender via a cache
// that can send data over notify.
type {{.Name}}CacheUpdater struct {
	{{.Name}}SenderHelper
	ctx context.Context
	key {{.KeyType}}
	cache *{{.Name}}Cache
}

func New{{.Name}}CacheUpdater(ctx context.Context, cache *{{.Name}}Cache, key {{.KeyType}}) *{{.Name}}CacheUpdater {
	s := &{{.Name}}CacheUpdater{
		ctx: ctx,
		key: key,
		cache: cache,
	}
	s.SetUpdater(s)
	return s
}

func (s *{{.Name}}CacheUpdater) Get() *{{.Name}} {
	obj := {{.Name}}{}
	if !s.cache.Get(&s.key, &obj) {
		obj.Key = s.key
	}
	return &obj
}

func (s *{{.Name}}CacheUpdater) Update(obj *{{.Name}}) error {
	s.cache.Update(s.ctx, obj, 0)
	return nil
}

type {{.Name}}SendAPI interface {
	Send(*{{.Name}}) error
}

// {{.Name}}SendUpdater implements {{.Name}}ObjectUpdater via a generic
// send API. To allow for building up the list of status messages
// which need to accumulate over time, we keep a local copy of
// the object.
type {{.Name}}SendUpdater struct {
	{{.Name}}SenderHelper
	ctx context.Context
	sender {{.Name}}SendAPI
	local {{.Name}}
	mux sync.Mutex
}

func New{{.Name}}SendUpdater(ctx context.Context, sender {{.Name}}SendAPI, key {{.KeyType}}) *{{.Name}}SendUpdater {
	s := &{{.Name}}SendUpdater{
		ctx: ctx,
		sender: sender,
	}
	s.local.Key = key
	s.SetUpdater(s)
	return s
}

func (s *{{.Name}}SendUpdater) Get() *{{.Name}} {
	s.mux.Lock()
	defer s.mux.Unlock()
	cp := {{.Name}}{}
	cp.DeepCopyIn(&s.local)
	return &cp
}

func (s *{{.Name}}SendUpdater) Update(obj *{{.Name}}) error {
	s.mux.Lock()
	s.local.DeepCopyIn(obj)
	s.mux.Unlock()
	return s.sender.Send(obj)
}

// {{.Name}}PrintUpdater just prints the updates
type {{.Name}}PrintUpdater struct {
	{{.Name}}SenderHelper
}

func New{{.Name}}PrintUpdater() *{{.Name}}PrintUpdater {
	s := &{{.Name}}PrintUpdater{}
	s.SetUpdater(s)
	return s
}

func (s *{{.Name}}PrintUpdater) Get() *{{.Name}} {
	return &{{.Name}}{}
}

func (s *{{.Name}}PrintUpdater) Update(obj *{{.Name}}) error {
	fmt.Printf("%v\n", obj)
	return nil
}

{{- end}}

{{if ne (.WaitForState) ("")}}
{{if eq (.WaitForState) ("TrackedState")}}
func WaitFor{{.Name}}(ctx context.Context, key *{{.KeyType}}, store {{.ParentObjName}}Store, targetState {{.WaitForState}}, transitionStates map[{{.WaitForState}}]struct{}, errorState {{.WaitForState}}, successMsg string, send func(*Result) error, crmMsgCh <-chan *redis.Message) error {
{{- else}}
func WaitFor{{.Name}}(ctx context.Context, key *{{.KeyType}}, store {{.Name}}Store, targetState distributed_match_engine.{{.WaitForState}}, transitionStates map[distributed_match_engine.{{.WaitForState}}]struct{}, errorState distributed_match_engine.{{.WaitForState}}, successMsg string, send func(*Result) error, crmMsgCh <-chan *redis.Message) error {
{{- end}}
	var lastMsgCnt int
	var err error

	handleTargetState := func() {
		{{- if eq (.WaitForState) ("TrackedState")}}
		if targetState == TrackedState_NOT_PRESENT {
			send(&Result{Message: {{.WaitForState}}_CamelName[int32(targetState)]})
		}
		{{- end}}
		if successMsg != "" && send != nil {
			send(&Result{Message: successMsg})
		}
	}

	// State updates come via Redis, since they are bundled with status updates.
	// However, the Redis channel is set up after the Etcd transaction to commit
	// the state change (i.e. CREATE_REQUESTED) in order to treat Etcd as the
	// source of truth for concurrent changes, so there is a small timing window
	// where the state may be updated by the info (from CRM) before the Redis
	// subscription is set up. So here our initial state needs to come from Etcd
	// in case both it and Redis were updated before the crmMsgCh was set up.
	{{- if eq (.WaitForState) ("TrackedState")}}
	curState := {{.WaitForState}}_NOT_PRESENT
	buf := {{.ParentObjName}}{}
	{{- else}}
	curState := distributed_match_engine.{{.WaitForState}}_CLOUDLET_STATE_NOT_PRESENT
	buf := {{.Name}}{}
	{{- end}}
	if store.Get(ctx, key, &buf) {
		curState = buf.State
	}
	if curState == targetState {
		handleTargetState()
		return nil
	}

    if crmMsgCh == nil {
		log.SpanLog(ctx, log.DebugLevelApi, "wait for {{.Name}} func missing crmMsgCh", "key", key)
		return fmt.Errorf("wait for {{.Name}} missing redis message channel")
	}

	for {
		select {
		case chObj := <-crmMsgCh:
			if chObj == nil {
				// Since msg chan is a receive-only chan, it will return nil if
				// connection to redis server is disrupted. But the object might
				// still be in progress. Hence, just show a message about the failure,
				// so that user can manually look at object's progress
				if send != nil {
					msg := fmt.Sprintf("Failed to get progress messages. Please use Show{{.ParentObjName}} to check current status")
					send(&Result{Message: msg})
				}
				return nil
			}
			info := {{.Name}}{}
			err = json.Unmarshal([]byte(chObj.Payload), &info)
			if err != nil {
				return err
			}
			curState = info.State
			log.SpanLog(ctx, log.DebugLevelApi, "Received crm update for {{.Name}}", "key", key, "obj", info)
			if send != nil {
				for ii := lastMsgCnt; ii < len(info.Status.Msgs); ii++ {
					send(&Result{Message: info.Status.Msgs[ii]})
				}
				lastMsgCnt = len(info.Status.Msgs)
			}

			switch info.State {
			case errorState:
				errs := strings.Join(info.Errors, ", ")
				if len(info.Errors) == 1 {
					err = fmt.Errorf("%s", errs)				
				} else {
					err = fmt.Errorf("Encountered failures: %s", errs)
				}
				return err
			case targetState:
				handleTargetState()
				return nil
			}
		case <-ctx.Done():
			if _, found := transitionStates[curState]; found {
				// no success response, but state is a valid transition
				// state. That means work is still in progress.
				// Notify user that this is not an error.
				// Do not undo since CRM is still busy.
				if send != nil {
					{{- if eq (.WaitForState) ("TrackedState")}}
					msg := fmt.Sprintf("Timed out while work still in progress state %s. Please use Show{{.ParentObjName}} to check current status", {{.WaitForState}}_CamelName[int32(curState)])
					{{- else}}
					msg := fmt.Sprintf("Timed out while work still in progress state %s. Please use Show{{.ParentObjName}} to check current status", distributed_match_engine.{{.WaitForState}}_CamelName[int32(curState)])
					{{- end}}
					send(&Result{Message: msg})
				}
				err = nil
			} else {
				err = fmt.Errorf("Timed out; expected state %s but is %s",
					{{- if eq (.WaitForState) ("TrackedState")}}
					{{.WaitForState}}_CamelName[int32(targetState)],
					{{.WaitForState}}_CamelName[int32(curState)])
					{{- else}}
					distributed_match_engine.{{.WaitForState}}_CamelName[int32(targetState)],
					distributed_match_engine.{{.WaitForState}}_CamelName[int32(curState)])
					{{- end}}
			}
			return err
		}
	}
}
{{- end}}

`

type sublistLookupTemplateArgs struct {
	Name       string
	KeyType    string
	LookupType string
	LookupName string
}

var sublistLookupTemplateIn = `
type {{.Name}}By{{.LookupName}} struct {
	{{.LookupName}}s map[{{.LookupType}}]map[{{.KeyType}}]struct{}
	Mux util.Mutex
}

func (s *{{.Name}}By{{.LookupName}}) Init() {
	s.{{.LookupName}}s = make(map[{{.LookupType}}]map[{{.KeyType}}]struct{})
}

func (s *{{.Name}}By{{.LookupName}}) Updated(old *{{.Name}}, new *{{.Name}}) map[{{.LookupType}}]struct{} {
	// the below func must be implemented by the user:
	// {{.Name}}.Get{{.LookupName}}s() map[{{.LookupType}}]struct{}
	old{{.LookupName}}s := make(map[{{.LookupType}}]struct{})
	if old != nil {
		old{{.LookupName}}s = old.Get{{.LookupName}}s()
	}
	new{{.LookupName}}s := new.Get{{.LookupName}}s()

	for lookup, _ := range old{{.LookupName}}s {
		if _, found := new{{.LookupName}}s[lookup]; found {
			delete(old{{.LookupName}}s, lookup)
			delete(new{{.LookupName}}s, lookup)
		}
	}

	s.Mux.Lock()
	defer s.Mux.Unlock()

	changed := make(map[{{.LookupType}}]struct{})
	for lookup, _ := range old{{.LookupName}}s {
		// remove
		s.removeRef(lookup, old.GetKeyVal())
		changed[lookup] = struct{}{}
	}
	for lookup, _ := range new{{.LookupName}}s {
		// add
		s.addRef(lookup, new.GetKeyVal())
		changed[lookup] = struct{}{}
	}
	return changed
}

func (s *{{.Name}}By{{.LookupName}}) Deleted(old *{{.Name}}) {
	old{{.LookupName}}s := old.Get{{.LookupName}}s()

	s.Mux.Lock()
	defer s.Mux.Unlock()

	for lookup, _ := range old{{.LookupName}}s {
		s.removeRef(lookup, old.GetKeyVal())
	}
}

func (s *{{.Name}}By{{.LookupName}}) addRef(lookup {{.LookupType}}, key {{.KeyType}}) {
	{{.KeyType}}s, found := s.{{.LookupName}}s[lookup]
	if !found {
		{{.KeyType}}s = make(map[{{.KeyType}}]struct{})
		s.{{.LookupName}}s[lookup] = {{.KeyType}}s
	}
	{{.KeyType}}s[key] = struct{}{}
}

func (s *{{.Name}}By{{.LookupName}}) removeRef(lookup {{.LookupType}}, key {{.KeyType}}) {
	{{.KeyType}}s, found := s.{{.LookupName}}s[lookup]
	if found {
		delete({{.KeyType}}s, key)
		if len({{.KeyType}}s) == 0 {
			delete(s.{{.LookupName}}s, lookup)
		}
	}
}

func (s *{{.Name}}By{{.LookupName}}) Find(lookup {{.LookupType}}) []{{.KeyType}} {
	s.Mux.Lock()
	defer s.Mux.Unlock()

	list := []{{.KeyType}}{}
	for k, _ := range s.{{.LookupName}}s[lookup] {
		list = append(list, k)
	}
	return list
}

func (s *{{.Name}}By{{.LookupName}}) HasRef(lookup {{.LookupType}}) bool {
	s.Mux.Lock()
	defer s.Mux.Unlock()

	_, found := s.{{.LookupName}}s[lookup]
	return found
}

// Convert to dumpable format. JSON cannot marshal maps with struct keys.
func (s *{{.Name}}By{{.LookupName}}) Dumpable() map[string]interface{} {
	s.Mux.Lock()
	defer s.Mux.Unlock()

	dat := make(map[string]interface{})
	for lookup, keys := range s.{{.LookupName}}s {
		keystrs := make(map[string]interface{})
		for k, _ := range keys {
			keystrs[k.GetKeyString()] = struct{}{}
		}
		dat[lookup.GetKeyString()] = keystrs
	}
	return dat
}

`

type subfieldLookupTemplateArgs struct {
	Name        string
	KeyType     string
	LookupType  string
	LookupName  string
	LookupField string
}

var subfieldLookupTemplateIn = `
type {{.Name}}By{{.LookupName}} struct {
	{{.LookupName}}s map[{{.LookupType}}]map[{{.KeyType}}]struct{}
	Mux util.Mutex
}

func (s *{{.Name}}By{{.LookupName}}) Init() {
	s.{{.LookupName}}s = make(map[{{.LookupType}}]map[{{.KeyType}}]struct{})
}

func (s *{{.Name}}By{{.LookupName}}) Updated(obj *{{.Name}}) {
	lookup := obj.{{.LookupField}}

	s.Mux.Lock()
	defer s.Mux.Unlock()

	{{.KeyType}}s, found := s.{{.LookupName}}s[lookup]
	if !found {
		{{.KeyType}}s = make(map[{{.KeyType}}]struct{})
		s.{{.LookupName}}s[lookup] = {{.KeyType}}s
	}
	{{.KeyType}}s[obj.GetKeyVal()] = struct{}{}
}

func (s *{{.Name}}By{{.LookupName}}) Deleted(obj *{{.Name}}) {
	lookup := obj.{{.LookupField}}

	s.Mux.Lock()
	defer s.Mux.Unlock()

	{{.KeyType}}s, found := s.{{.LookupName}}s[lookup]
	if found {
		delete({{.KeyType}}s, obj.GetKeyVal())
		if len({{.KeyType}}s) == 0 {
			delete(s.{{.LookupName}}s, lookup)
		}
	}
}

func (s *{{.Name}}By{{.LookupName}}) Find(lookup {{.LookupType}}) []{{.KeyType}} {
	s.Mux.Lock()
	defer s.Mux.Unlock()

	list := []{{.KeyType}}{}
	for k, _ := range s.{{.LookupName}}s[lookup] {
		list = append(list, k)
	}
	return list
}

`

type keysTemplateArgs struct {
	Name             string
	KeyType          string
	ObjAndKey        bool
	StreamKey        bool
	StringKeyField   string
	StringKeyFieldLC string
	IsMessage        bool
	HasMessageId     bool
	HasMoreReplies   bool
}

var keysTemplateIn = `
{{- if .StringKeyField}}
type {{.Name}}Key string

func (k {{.Name}}Key) GetKeyString() string {
	return string(k)
}

func {{.Name}}KeyStringParse(str string, key *{{.Name}}Key) {
	*key = {{.Name}}Key(str)
}

func (k {{.Name}}Key) NotFoundError() error {
	return fmt.Errorf("{{.Name}} key %s not found", k.GetKeyString())
}

func (k {{.Name}}Key) ExistsError() error {
	return fmt.Errorf("{{.Name}} key %s already exists", k.GetKeyString())
}

func (k {{.Name}}Key) BeingDeletedError() error {
	return fmt.Errorf("{{.Name}} key %s is being deleted", k.GetKeyString())
}

func (k {{.Name}}Key) GetTags() map[string]string {
	return map[string]string{
		"{{.StringKeyFieldLC}}": string(k),
	}
}

func (k {{.Name}}Key) AddTagsByFunc(addTag AddTagFunc) {
	addTag("{{.StringKeyFieldLC}}", string(k))
}

func (k {{.Name}}Key) AddTags(tags map[string]string) {
	tagMap := TagMap(tags)
	k.AddTagsByFunc(tagMap.AddTag)
}
{{- end}}

func (m *{{.Name}}) GetObjKey() objstore.ObjKey {
	return m.GetKey()
}

func (m *{{.Name}}) GetKey() *{{.KeyType}} {
{{- if .ObjAndKey}}
	return m
{{- else if (.StringKeyField)}}
	key := {{.Name}}Key(m.{{.StringKeyField}})
	return &key
{{- else}}
	return &m.Key
{{- end}}
}

func (m *{{.Name}}) GetKeyVal() {{.KeyType}} {
{{- if .ObjAndKey}}
	return *m
{{- else if (.StringKeyField)}}
	return {{.Name}}Key(m.{{.StringKeyField}})
{{- else}}
	return m.Key
{{- end}}
}

func (m *{{.Name}}) SetKey(key *{{.KeyType}}) {
{{- if .ObjAndKey}}
	*m = *key
{{- else if (.StringKeyField)}}
	m.{{.StringKeyField}} = string(*key)
{{- else}}
	m.Key = *key
{{- end}}
}

func CmpSort{{.Name}}(a {{.Name}}, b {{.Name}}) bool {
{{- if .ObjAndKey}}
	return a.GetKeyString() < b.GetKeyString()
{{- else if (.StringKeyField)}}
	return a.{{.StringKeyField}} < b.{{.StringKeyField}}
{{- else}}
	return a.Key.GetKeyString() < b.Key.GetKeyString()
{{- end}}
}

{{- if .StreamKey}}
func (m *{{.KeyType}}) StreamKey() string {
	return fmt.Sprintf("{{.Name}}StreamKey: %s", m.String())
}
{{- end}}

{{- if .IsMessage}}
// MessageKey can be used as a channel name which includes the
// key value for pubsub, to listen for this specific object type
// plus key value.
func (m *{{.Name}}) MessageKey() string {
	return fmt.Sprintf("msg/key/{{.Name}}/%s", m.GetKey().GetKeyString())
}
{{- end}}


`

type ugpradeError struct {
	CurHash        string
	CurHashEnumVal int32
	NewHash        string
	NewHashEnumVal int32
}

var upgradeErrorTemplete = `
======WARNING=======
Current data model hash({{.NewHash}}) doesn't match the latest supported one({{.CurHash}}).
This is due to an upsupported change in the key of some objects in a .proto file.
In order to ensure a smooth upgrade for the production environment please make sure to add the following to version.proto file:

enum VersionHash {
	...
	{{.CurHash}} = {{.CurHashEnumVal}};
	{{.NewHash}} = {{.NewHashEnumVal}} [(protogen.upgrade_func) = "sample_upgrade_function"]; <<<===== Add this line
	...
}

IMPORTANT: The field value {{.NewHashEnumVal}} must be a monotonically increasing
value and must never be reused.

Implementation of "sample_upgrade_function" should be added tp pkg/controller/upgrade_funcs.go

NOTE: If no upgrade function is needed don't need to add "[(protogen.upgrade_func) = "sample_upgrade_function];" to
the VersionHash enum.

A unit test data for the automatic unit test of the upgrade function should be added to pkg/controller/upgrade_testfiles
   - PreUpgradeData - what key/value objects are trying to be upgraded
   - PostUpgradeData - what the resulting object store should look like
====================
`

func (m *mex) generateMessage(file *generator.FileDescriptor, desc *generator.Descriptor) {
	message := desc.DescriptorProto
	if GetGenerateMatches(message) && message.Field != nil {
		m.P("func (m *", message.Name, ") Matches(o *", message.Name, ", fopts ...MatchOpt) bool {")
		m.P("opts := MatchOptions{}")
		m.P("applyMatchOptions(&opts, fopts...)")
		m.P("if o == nil {")
		m.P("if opts.Filter { return true }")
		m.P("return false")
		m.P("}")
		for ii, field := range message.Field {
			if ii == 0 && *field.Name == "fields" {
				continue
			}
			m.generateFieldMatches(message, field)
		}
		m.P("return true")
		m.P("}")
		m.P("")
	}
	if gensupport.HasGrpcFields(message) {
		m.generateFields([]string{*message.Name + "Field"}, []string{}, desc)
		m.P("")
		m.P("var ", *message.Name, "AllFields = []string{")
		m.generateAllFields(AllFieldsGenSlice, []string{*message.Name + "Field"}, []string{}, desc)
		m.P("}")
		m.P("")
		m.P("var ", *message.Name, "AllFieldsMap = NewFieldMap(map[string]struct{}{")
		m.generateAllFields(AllFieldsGenMap, []string{*message.Name + "Field"}, []string{}, desc)
		m.P("})")
		m.P("")
		m.P("var ", *message.Name, "AllFieldsStringMap = map[string]string{")
		m.generateAllStringFieldsMap(AllFieldsGenMap, []string{}, []string{}, *message.Name+"Field", desc)
		m.P("}")
		m.P("")
		m.P("func (m *", *message.Name, ") IsKeyField(s string) bool {")
		m.generateIsKeyField([]string{}, []string{*message.Name + "Field"}, desc)
		m.P("}")
		m.P("")
		m.P("func (m *", message.Name, ") DiffFields(o *", message.Name, ", fields *FieldMap) {")
		m.generateDiffFields([]string{}, []string{*message.Name + "Field"}, desc)
		m.P("}")
		m.P("")
		m.P("func (m *", message.Name, ") GetDiffFields(o *", message.Name, ") *FieldMap {")
		m.P("diffFields := NewFieldMap(nil)")
		m.P("m.DiffFields(o, diffFields)")
		m.P("return diffFields")
		m.P("}")
		m.P("")
		for _, service := range file.Service {
			if *service.Name != *message.Name+"Api" {
				continue
			}
			if len(service.Method) == 0 {
				continue
			}
			for _, method := range service.Method {
				if gensupport.GetCamelCasePrefix(*method.Name) != "Update" {
					continue
				}
				noconfigMap := make(map[string]struct{})
				m.P("var ", *method.Name, "FieldsMap = NewFieldMap(map[string]struct{}{")
				fieldPrefix := *message.Name + "Field"
				m.generateMethodFields(fieldPrefix, []string{fieldPrefix}, noconfigMap, desc, method)
				m.P("})")
				m.P("")
				args := cudTemplateArgs{
					Name: *message.Name,
				}
				m.fieldsValTemplate.Execute(m.gen.Buffer, args)
				break
			}
		}
	}

	if desc.GetOptions().GetMapEntry() {
		return
	}

	msgtyp := m.gen.TypeName(desc)
	m.P("func (m *", msgtyp, ") Clone() *", msgtyp, " {")
	m.P("cp := &", msgtyp, "{}")
	m.P("cp.DeepCopyIn(m)")
	m.P("return cp")
	m.P("}")
	m.P()

	m.generateListAddRemoves(make([]string, 0), desc, desc, make([]*generator.Descriptor, 0))

	if GetGenerateCopyInFields(message) {
		m.P("func (m *", msgtyp, ") CopyInFields(src *", msgtyp, ") int {")
		if m.needsUpdateListActionVar(desc, make([]*generator.Descriptor, 0)) {
			if gensupport.FindField(message, gensupport.UpdateListActionField) != nil {
				m.P("updateListAction := src.", gensupport.UpdateListActionField)
			} else {
				m.P("updateListAction := \"", util.UpdateListActionReplace, "\"")
			}
		}
		m.P("changed := 0")
		if gensupport.HasGrpcFields(message) {
			m.P("fmap := MakeFieldMap(src.Fields)")
		}
		m.generateCopyIn(make([]string, 0), make([]string, 0), desc, make([]*generator.Descriptor, 0), gensupport.HasGrpcFields(message))
		m.P("return changed")
		m.P("}")
		m.P("")
	}

	m.generateDeepCopyIn(desc)

	if GetGenerateCud(message) {
		keyType, err := m.support.GetMessageKeyType(m.gen, desc)
		if err != nil {
			m.gen.Fail(err.Error())
		}
		args := cudTemplateArgs{
			Name:      *message.Name,
			CudName:   *message.Name + "Cud",
			HasFields: gensupport.HasGrpcFields(message),
			ObjAndKey: gensupport.GetObjAndKey(message),
			KeyType:   keyType,
		}
		if m.refData == nil {
			panic("empty ref data")
		}
		_ = len(m.refData.RefTos)
		if refToGroup, ok := m.refData.RefTos[*message.Name]; ok {
			for _, byObjField := range refToGroup.Bys {
				name := byObjField.By.Type + strings.Replace(byObjField.Field.HierName, ".", "", -1)
				args.RefBys = append(args.RefBys, name)
			}
		}
		m.cudTemplate.Execute(m.gen.Buffer, args)
		m.importLog = true
	}
	if GetGenerateCache(message) {
		keyType, err := m.support.GetMessageKeyType(m.gen, desc)
		if err != nil {
			m.gen.Fail(err.Error())
		}
		args := cacheTemplateArgs{
			Name:           *message.Name,
			CudCache:       GetGenerateCud(message),
			NotifyCache:    GetNotifyCache(message),
			NotifyFlush:    GetNotifyFlush(message),
			ParentObjName:  GetParentObjName(message),
			WaitForState:   GetGenerateWaitForState(message),
			ObjAndKey:      gensupport.GetObjAndKey(message),
			CustomKeyType:  gensupport.GetCustomKeyType(message),
			KeyType:        keyType,
			StreamOut:      gensupport.GetGenerateCudStreamout(message),
			StringKeyField: gensupport.GetStringKeyField(message),
		}
		stateField := gensupport.FindField(message, "State")
		if stateField != nil {
			args.StateFieldType = m.support.GoType(m.gen, stateField)
		}
		m.cacheTemplate.Execute(m.gen.Buffer, args)
		m.importUtil = true
		m.importLog = true
		if args.WaitForState != "" {
			m.importErrors = true
			m.importStrings = true
			m.importRedis = true
		}
		if args.ParentObjName != "" {
			m.importSync = true
		}
		m.generateUsesOrg(message)
	}
	if lookups := GetGenerateLookupBySublist(message); lookups != "" {
		keyType, err := m.support.GetMessageKeyType(m.gen, desc)
		if err != nil {
			m.gen.Fail(err.Error())
		}
		list := strings.Split(lookups, ",")
		for _, lookup := range list {
			lookup = strings.TrimSpace(lookup)
			nameType := strings.Split(lookup, ":")
			args := sublistLookupTemplateArgs{
				Name:       *message.Name,
				KeyType:    keyType,
				LookupType: nameType[0],
				LookupName: nameType[0],
			}
			if len(nameType) > 1 {
				args.LookupName = nameType[1]
			}
			m.sublistLookupTemplate.Execute(m.gen.Buffer, args)
			m.importUtil = true
			m.importJson = true
		}
	}
	if lookups := GetGenerateLookupBySubfield(message); lookups != "" {
		keyType, err := m.support.GetMessageKeyType(m.gen, desc)
		if err != nil {
			m.gen.Fail(err.Error())
		}
		list := strings.Split(lookups, ",")
		for _, lookup := range list {
			lookup = strings.TrimSpace(lookup)
			_, field, err := gensupport.FindHierField(m.gen, message, lookup)
			if err != nil {
				m.gen.Fail(err.Error())
				continue
			}
			args := subfieldLookupTemplateArgs{
				Name:        *message.Name,
				KeyType:     keyType,
				LookupType:  m.support.GoType(m.gen, field),
				LookupName:  m.support.GoType(m.gen, field),
				LookupField: lookup,
			}
			m.subfieldLookupTemplate.Execute(m.gen.Buffer, args)
			m.importUtil = true
		}
	}
	if GetObjKey(message) || gensupport.GetObjAndKey(message) {
		// this is a key object
		m.P("func (m *", message.Name, ") GetKeyString() string {")
		m.P("key, err := json.Marshal(m)")
		m.P("if err != nil {")
		m.P("log.FatalLog(\"Failed to marshal ", message.Name, " key string\", \"obj\", m)")
		m.P("}")
		m.P("return string(key)")
		m.P("}")
		m.P("")

		m.P("func ", message.Name, "StringParse(str string, key *", message.Name, ") {")
		m.P("err := json.Unmarshal([]byte(str), key)")
		m.P("if err != nil {")
		m.P("log.FatalLog(\"Failed to unmarshal ", message.Name, " key string\", \"str\", str)")
		m.P("}")
		m.P("}")
		m.P("")

		m.P("func (m *", message.Name, ") NotFoundError() error {")
		m.P("return fmt.Errorf(\"", strings.TrimSuffix(*message.Name, "Key"), " key %s not found\", m.GetKeyString())")
		m.P("}")
		m.P("")

		m.P("func (m *", message.Name, ") ExistsError() error {")
		m.P("return fmt.Errorf(\"", strings.TrimSuffix(*message.Name, "Key"), " key %s already exists\", m.GetKeyString())")
		m.P("}")
		m.P("")

		m.P("func (m *", message.Name, ") BeingDeletedError() error {")
		m.P("return fmt.Errorf(\"", strings.TrimSuffix(*message.Name, "Key"), " %s is being deleted\", m.GetKeyString())")
		m.P("}")
		m.P("")

		hasKeyTags := false
		for _, field := range message.Field {
			if field.Type == nil || field.OneofIndex != nil {
				continue
			}
			if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				continue
			}
			tag := GetKeyTag(field)
			if tag == "" {
				m.gen.Fail(*message.Name, "field", *field.Name, "missing protogen.keytag")
			}
			fname := generator.CamelCase(*field.Name)
			m.P("var ", message.Name, "Tag", fname, " = \"", tag, "\"")
			hasKeyTags = true
		}
		if hasKeyTags {
			m.P()
		}
		m.P("func (m *", message.Name, ") GetTags() map[string]string {")
		m.P("tags := make(map[string]string)")
		m.P("m.AddTags(tags)")
		m.P("return tags")
		m.P("}")
		m.P()

		m.P("func (m *", message.Name, ") AddTagsByFunc(addTag AddTagFunc) {")
		m.setKeyTags([]string{}, desc, []*generator.Descriptor{})
		m.P("}")
		m.P()

		m.P("func (m *", message.Name, ") AddTags(tags map[string]string) {")
		m.P("tagMap := TagMap(tags)")
		m.P("m.AddTagsByFunc(tagMap.AddTag)")
		m.P("}")
		m.P()

		m.importJson = true
		m.importLog = true
	}

	if gensupport.GetMessageKey(message) != nil || gensupport.GetObjAndKey(message) || gensupport.GetStringKeyField(message) != "" {
		// this is an object that has a key field
		keyType, err := m.support.GetMessageKeyType(m.gen, desc)
		if err != nil {
			m.gen.Fail(err.Error())
		}
		args := keysTemplateArgs{
			Name:           *message.Name,
			KeyType:        keyType,
			ObjAndKey:      gensupport.GetObjAndKey(message),
			StreamKey:      GetGenerateStreamKey(message),
			StringKeyField: gensupport.GetStringKeyField(message),
			IsMessage:      GetNotifyMessage(message),
		}
		args.StringKeyFieldLC = strings.ToLower(args.StringKeyField)
		m.keysTemplate.Execute(m.gen.Buffer, args)
	}
	if GetNotifyMessage(message) {
		m.P("func (m *", message.Name, ") MessageTypeKey() string {")
		m.P("return \"msg/type/", message.Name, "\"")
		m.P("}")
		m.P()
	}

	//Generate enum values validation
	m.generateEnumValidation(message, desc)
	m.generateClearTagged(desc)
	m.generateClearRedisOnlyFields(desc)

	visited := make([]*generator.Descriptor, 0)
	if gensupport.HasHideTags(m.gen, desc, protogen.E_Hidetag, visited) {
		m.generateHideTags(desc)
	}

	if gensupport.GetE2edata(message) {
		m.P("func (m *", message.Name, ") IsEmpty() bool {")
		for _, field := range message.Field {
			if field.Type == nil || field.OneofIndex != nil {
				continue
			}
			if *field.Type != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				continue
			}
			fname := generator.CamelCase(*field.Name)
			m.P("if m.", fname, " != nil {")
			m.P("return false")
			m.P("}")
		}
		m.P("return true")
		m.P("}")
		m.P()
		storeReadFields := []*descriptor.FieldDescriptorProto{}
		for _, field := range message.Field {
			if field.Type == nil || field.OneofIndex != nil {
				continue
			}
			if *field.Type != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				continue
			}
			fieldDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			if !GetGenerateCud(fieldDesc.DescriptorProto) {
				continue
			}
			storeReadFields = append(storeReadFields, field)
		}
		if len(storeReadFields) > 0 {
			m.P("func (m *", message.Name, ") StoreRead(ctx context.Context, kvstore objstore.KVStore) error {")
			for _, field := range storeReadFields {
				fieldDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
				inType := *fieldDesc.DescriptorProto.Name
				fname := generator.CamelCase(*field.Name)
				m.P(field.Name, ", err := StoreList", inType, "(ctx, kvstore)")
				m.P("if err != nil { return err }")
				if GetSingularData(fieldDesc.DescriptorProto) {
					m.P("if len(", field.Name, ") > 0 {")
					m.P("m.", fname, " = &", field.Name, "[0]")
					m.P("}")
				} else {
					m.P("m.", fname, " = ", field.Name)
				}
			}
			m.P("return nil")
			m.P("}")
			m.P()
			m.importContext = true
			m.importObjstore = true
		}
	}
}

func (m *mex) generateVersionString(hashStr string, ID int32) {
	m.P("// Keys being hashed:")
	for _, v := range m.keyMessages {
		m.P("// ", v.Name)
	}
	m.P()
	m.P("type DataModelVersion struct {")
	m.P("Hash string")
	m.P("ID int32")
	m.P("}")
	m.P()
	m.P("func GetDataModelVersion() *DataModelVersion {")
	m.P("return &DataModelVersion{")
	m.P("Hash: \"", hashStr, "\",")
	m.P("ID: ", int(ID), ",")
	m.P("}")
	m.P("}")
}

func validateVersionHash(latestVer *descriptor.EnumValueDescriptorProto, hashStr string, file *generator.FileDescriptor) {
	// We need to check the hash and verify that we have the correct one
	// If we don't have a correct one fail suggesting an upgrade function
	// Check the last one(it's the latest) and if it doesn't match fail
	// Check the substring of the value
	if !strings.Contains(*latestVer.Name, hashStr) {
		var upgradeTemplate *template.Template
		upgradeTemplate = template.Must(template.New("upgrade").Parse(upgradeErrorTemplete))
		buf := bytes.Buffer{}
		upgErr := ugpradeError{
			CurHash:        *latestVer.Name,
			CurHashEnumVal: *latestVer.Number,
			NewHash:        "HASH_" + hashStr,
			NewHashEnumVal: *latestVer.Number + 1,
		}
		if err := upgradeTemplate.Execute(&buf, &upgErr); err != nil {
			log.Fatalf("Cannot execute upgrade error template %v\n", err)
		}
		log.Fatalf("%s", buf.String())
	}
}

// Subset of the FieldDescriptorProto that is used to identify whether we need to trigger an
// incompatible upgrade or not
type FieldDescriptorProtoHashable struct {
	Name     *string
	Number   *int32
	Label    *descriptor.FieldDescriptorProto_Label
	Type     *descriptor.FieldDescriptorProto_Type
	TypeName *string
	Extendee *string
}

// Unique idenitifable message object, which is used in a version hash calculation
type HashableKey struct {
	Name  *string
	Field []FieldDescriptorProtoHashable
}

// This function generates an array of HashableKey[s] from an array of the DescriptorProto
// messages. HashableKey is defined to keep track of only the specific sets of
// fields of the DescriptorProto which make it unique
// NOTE: There is a possiblity that some of the sub-strucutres of the key messages
// is not an obj_key itself and we might miss it in a hash calculation.
// If this ever becomes a problem we should make sure to track all the sub-structs that
// are not key_obj
func getHashObjsFromMsgs(msgs []descriptor.DescriptorProto) []HashableKey {
	objs := make([]HashableKey, 0)
	for _, m := range msgs {
		o := HashableKey{
			Name: m.Name,
		}
		for _, dp := range m.Field {
			dpHash := FieldDescriptorProtoHashable{
				Name:     dp.Name,
				Number:   dp.Number,
				Label:    dp.Label,
				Type:     dp.Type,
				TypeName: dp.TypeName,
				Extendee: dp.Extendee,
			}
			o.Field = append(o.Field, dpHash)
		}
		objs = append(objs, o)
	}
	return objs
}

// Hash function for the Data Model Version
func getKeyVersionHash(msgs []descriptor.DescriptorProto, salt string) [16]byte {
	// Sort the messages to make sure we are generate repeatable hash
	sort.Slice(msgs, func(i, j int) bool {
		return *msgs[i].Name < *msgs[j].Name
	})
	// Need to build an array of HashableKeys from msgs
	hashObjs := getHashObjsFromMsgs(msgs)
	arrBytes := []byte{}
	for _, o := range hashObjs {
		jsonBytes, _ := json.Marshal(o)
		arrBytes = append(arrBytes, jsonBytes...)
	}
	// add salt
	arrBytes = append(arrBytes, []byte(salt)...)
	return md5.Sum(arrBytes)

}

// Generate a single check for an enum
func (m *mex) generateEnumCheck(field *descriptor.FieldDescriptorProto, elem string) {
	m.P("if _, ok := ", m.support.GoType(m.gen, field), "_name[int32(", elem,
		")]; !ok {")
	m.P("return errors.New(\"invalid ", generator.CamelCase(*field.Name),
		"\")")
	m.P("}")
	m.importErrors = true
}

func (m *mex) generateMessageEnumCheck(elem string) {
	m.P("if err := ", elem, ".ValidateEnums(); err != nil {")
	m.P("return err")
	m.P("}")
}

// Generate enum validation method for each message
// NOTE: we don't check for set fields. This is ok as
// long as enums start at 0 and unset fields are zeroed out
func (m *mex) generateEnumValidation(message *descriptor.DescriptorProto, desc *generator.Descriptor) {
	m.P("// Helper method to check that enums have valid values")
	if gensupport.HasGrpcFields(message) {
		m.P("// NOTE: ValidateEnums checks all Fields even if some are not set")
	}
	msgtyp := m.gen.TypeName(desc)
	m.P("func (m *", msgtyp, ") ValidateEnums() error {")
	for _, field := range message.Field {
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			// could be an array of enums
			if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
				m.P("for _, e := range m.", generator.CamelCase(*field.Name), " {")
				m.generateEnumCheck(field, "e")
				m.P("}")
			} else {
				m.generateEnumCheck(field, "m."+generator.CamelCase(*field.Name))
			}
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			// Don't try to generate a call to a vlidation for external package
			if _, ok := m.support.MessageTypesGen[field.GetTypeName()]; !ok {
				continue
			}
			// Not supported OneOf types
			if field.OneofIndex != nil {
				continue
			}
			// could be an array of messages
			if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
				m.P("for _, e := range m.", generator.CamelCase(*field.Name), " {")
				m.generateMessageEnumCheck("e")
				m.P("}")
			} else if gogoproto.IsNullable(field) {
				m.P("if m.", generator.CamelCase(*field.Name), " != nil {")
				m.generateMessageEnumCheck("m." + generator.CamelCase(*field.Name))
				m.P("}")
			} else {
				m.generateMessageEnumCheck("m." + generator.CamelCase(*field.Name))
			}
		}
	}
	m.P("return nil")
	m.P("}")
	m.P("")
}

func (m *mex) generateHideTags(desc *generator.Descriptor) {
	msgName := desc.DescriptorProto.Name
	m.P("func Ignore", msgName, "Fields(taglist string) cmp.Option {")
	m.P("names := []string{}")
	m.P("tags := make(map[string]struct{})")
	m.P("for _, tag := range strings.Split(taglist, \",\") {")
	m.P("tags[tag] = struct{}{}")
	m.P("}")
	visited := make([]*generator.Descriptor, 0)
	m.generateHideTagFields(make([]string, 0), desc, visited)
	m.P("return cmpopts.IgnoreFields(", msgName, "{}, names...)")
	m.P("}")
	m.P()
	m.importStrings = true
	m.importCmp = true
}

func (m *mex) generateHideTagFields(parents []string, desc *generator.Descriptor, visited []*generator.Descriptor) {
	if gensupport.WasVisited(desc, visited) {
		return
	}
	msg := desc.DescriptorProto
	for _, field := range msg.Field {
		if field.Type == nil || field.OneofIndex != nil {
			continue
		}
		tag := GetHideTag(field)
		if tag == "" && *field.Type != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			continue
		}
		name := generator.CamelCase(*field.Name)
		hierField := strings.Join(append(parents, name), ".")

		if tag != "" {
			m.P("if _, found := tags[\"", tag, "\"]; found {")
			m.P("names = append(names, \"", hierField, "\")")
			m.P("}")
		}
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.generateHideTagFields(append(parents, name),
				subDesc, append(visited, desc))
		}
	}
}

func (m *mex) generateClearTagged(desc *generator.Descriptor) {
	msgName := strings.Join(desc.TypeName(), "_")
	m.P("func (s *", msgName, ") ClearTagged(tags map[string]struct{}) {")
	visited := make([]*generator.Descriptor, 0)
	srcPkg := m.support.GetPackageName(m.gen, desc)
	m.generateClearTaggedFields(srcPkg, make([]string, 0), desc, visited)
	m.P("}")
	m.P()
}

func (m *mex) generateClearTaggedFields(srcPkg string, parents []string, desc *generator.Descriptor, visited []*generator.Descriptor) {
	if gensupport.WasVisited(desc, visited) {
		return
	}
	msg := desc.DescriptorProto
	for _, field := range msg.Field {
		if field.Type == nil || field.OneofIndex != nil {
			continue
		}
		name := generator.CamelCase(*field.Name)
		hierField := strings.Join(append(parents, name), ".")
		mapType := m.support.GetMapType(m.gen, field)
		repeated := *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
		tag := GetHideTag(field)
		if tag == "" && *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE && mapType == nil {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			if srcPkg == m.support.GetPackageName(m.gen, subDesc) {
				// sub message should have ClearTags() defined
				nilCheck := false
				if repeated || gogoproto.IsNullable(field) {
					m.P("if s.", hierField, " != nil {")
					nilCheck = true
				}
				if repeated {
					m.P("for ii := 0; ii < len(s.", hierField, "); ii++ {")
					m.P("s.", hierField, "[ii].ClearTagged(tags)")
					m.P("}")
				} else {
					m.P("s.", hierField, ".ClearTagged(tags)")
				}
				if nilCheck {
					m.P("}")
				}
			} else {
				// recurse
				m.generateClearTaggedFields(srcPkg, append(parents, name), subDesc, append(visited, desc))
			}
			continue
		}
		if tag == "" {
			continue
		}
		m.P("if _, found := tags[\"", tag, "\"]; found {")

		// clear field
		nilval := "0"
		if repeated || mapType != nil {
			nilval = "nil"
		} else {
			switch *field.Type {
			case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
				if gogoproto.IsNullable(field) {
					nilval = "nil"
				} else {
					subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
					nilval = m.support.FQTypeName(m.gen, subDesc) + "{}"
				}
			case descriptor.FieldDescriptorProto_TYPE_STRING:
				nilval = "\"\""
			case descriptor.FieldDescriptorProto_TYPE_BOOL:
				nilval = "false"
			}
		}
		m.P("s.", hierField, " = ", nilval)
		m.P("}")
	}
}

func (m *mex) generateClearRedisOnlyFields(desc *generator.Descriptor) {
	msgName := strings.Join(desc.TypeName(), "_")
	msg := desc.DescriptorProto
	redisOnlyFieldExists := false
	for _, field := range msg.Field {
		RedisOnly := GetRedisOnly(field)
		if RedisOnly {
			redisOnlyFieldExists = true
			break
		}
	}

	if !redisOnlyFieldExists {
		return
	}

	m.P("func (s *", msgName, ") ClearRedisOnlyFields() {")
	m.P("// Clear fields so that they are not stored in DB, as they are cached in Redis")

	for _, field := range msg.Field {
		RedisOnly := GetRedisOnly(field)
		if !RedisOnly {
			continue
		}
		name := generator.CamelCase(*field.Name)
		mapType := m.support.GetMapType(m.gen, field)
		repeated := *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
		// clear field
		nilval := "0"
		if repeated || mapType != nil {
			nilval = "nil"
		} else {
			switch *field.Type {
			case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
				if gogoproto.IsNullable(field) {
					nilval = "nil"
				} else {
					subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
					nilval = m.support.FQTypeName(m.gen, subDesc) + "{}"
				}
			case descriptor.FieldDescriptorProto_TYPE_STRING:
				nilval = "\"\""
			case descriptor.FieldDescriptorProto_TYPE_BOOL:
				nilval = "false"
			}
		}
		m.P("s.", name, " = ", nilval)
	}
	m.P("}")
	m.P()
}

func (m *mex) setKeyTags(parents []string, desc *generator.Descriptor, visited []*generator.Descriptor) {
	for _, field := range desc.DescriptorProto.Field {
		if field.Type == nil || field.OneofIndex != nil {
			continue
		}
		name := generator.CamelCase(*field.Name)
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			subDesc := gensupport.GetDesc(m.gen, field.GetTypeName())
			m.setKeyTags(append(parents, name),
				subDesc, append(visited, desc))
			continue
		}
		tag := GetKeyTag(field)
		hierField := strings.Join(append(parents, name), ".")
		val := "m." + hierField
		if *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM {
			val = m.support.GoType(m.gen, field) + "_name[int32(" + val + ")]"
		}
		m.P("addTag(\"", tag, "\",", val, ")")
	}
}

func (m *mex) generateEnumDecodeHook() {
	m.P("// DecodeHook for use with the mapstructure package.")
	m.P("// Allows decoding to handle protobuf enums that are")
	m.P("// represented as strings.")
	m.P("func EnumDecodeHook(from, to reflect.Type, data interface{}) (interface{}, error) {")
	m.P("switch to {")
	for _, file := range m.gen.Request.ProtoFile {
		if !m.support.GenFile(*file.Name) {
			continue
		}
		for _, en := range file.EnumType {
			m.P("case reflect.TypeOf(", en.Name, "(0)):")
			m.P("return Parse", en.Name, "(data)")
		}
	}
	m.P("}")
	m.P("return data, nil")
	m.P("}")
	m.P()

	m.P("// GetEnumParseHelp gets end-user specific messages for ")
	m.P("// enum parse errors.")
	m.P("// It returns the enum type name, a help message with")
	m.P("// valid values, and a bool that indicates if a type was matched.")
	m.P("func GetEnumParseHelp(t reflect.Type) (string, string, bool) {")
	m.P("switch t {")
	for _, file := range m.gen.Request.ProtoFile {
		if !m.support.GenFile(*file.Name) {
			continue
		}
		for _, en := range file.EnumType {
			commonPrefix := gensupport.GetEnumCommonPrefix(en)
			validStrs := []string{}
			validInts := []string{}
			for _, val := range en.Value {
				validStrs = append(validStrs, strings.TrimPrefix(util.CamelCase(*val.Name), commonPrefix))
				validInts = append(validInts, strconv.Itoa(int(*val.Number)))
			}
			help := fmt.Sprintf(", valid values are one of %s, or %s", strings.Join(validStrs, ", "), strings.Join(validInts, ", "))
			m.P("case reflect.TypeOf(", en.Name, "(0)):")
			m.P("return \"", en.Name, "\", \"", help, "\", true")
		}
	}
	m.P("}")
	m.P("return \"\", \"\", false")
	m.P("}")
	m.P()

	m.importReflect = true
}

func (m *mex) generateShowCheck() {
	m.P("var ShowMethodNames = map[string]struct{}{")
	for _, file := range m.gen.Request.ProtoFile {
		if !m.support.GenFile(*file.Name) {
			continue
		}
		if len(file.Service) == 0 {
			continue
		}
		for _, service := range file.Service {
			if len(service.Method) == 0 {
				continue
			}
			for _, method := range service.Method {
				if gensupport.IsShow(method) {
					m.P("\"", method.Name, "\": struct{}{},")
				}
			}
		}
	}
	m.P("}")
	m.P()
	m.P("func IsShow(cmd string) bool {")
	m.P("_, found := ShowMethodNames[cmd]")
	m.P("return found")
	m.P("}")
	m.P()
}

func (m *mex) generateAllKeyTags() {
	tags := make(map[string]string)
	for _, file := range m.gen.Request.ProtoFile {
		if !m.support.GenFile(*file.Name) {
			continue
		}
		for _, message := range file.MessageType {
			for _, field := range message.Field {
				if field.Type == nil || field.OneofIndex != nil {
					continue
				}
				if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					continue
				}
				tag := GetKeyTag(field)
				if tag == "" {
					continue
				}
				if GetSkipKeyTagConflictCheck(field) {
					continue
				}

				fname := generator.CamelCase(*field.Name)
				tagLoc := *message.Name + "." + fname
				if conflict, found := tags[tag]; found {
					m.gen.Fail("KeyTag conflict for", tag, "between", tagLoc, "and", conflict)
				}
				tags[tag] = tagLoc
			}
		}
	}
	if len(tags) == 0 {
		return
	}
	list := []string{}
	for t, _ := range tags {
		list = append(list, t)
	}
	sort.Strings(list)
	m.P("var AllKeyTags = []string{")
	for _, tag := range list {
		m.P(`"`, tag, `",`)
	}
	m.P("}")
	m.P()
	m.P("var AllKeyTagsMap = map[string]struct{}{")
	for _, tag := range list {
		m.P(`"`, tag, `": struct{}{},`)
	}
	m.P("}")
	m.P()
}

func (m *mex) generateUsesOrg(message *descriptor.DescriptorProto) {
	usesOrg := GetUsesOrg(message)
	if usesOrg == "" {
		m.gen.Fail(*message.Name, "protogen.generate_cache option also requires protogen.uses_org option")
	}
	if usesOrg == "custom" {
		return
	}
	m.P()
	m.P("func (c *", message.Name, "Cache) UsesOrg(org string) bool {")
	if usesOrg == "none" {
		m.P("return false")
		m.P("}")
		return
	}
	keyIter := "_"
	valIter := "_"
	usesChecks := strings.Split(usesOrg, ",")
	kvChecks := [][]string{}
	for _, check := range usesChecks {
		kv := strings.Split(check, "=")
		if len(kv) != 2 {
			m.gen.Fail(*message.Name, "invalid uses_org check spec, expected a=b but was ", check)
			continue
		}
		if kv[0] == "key" {
			keyIter = "key"
		} else if kv[0] == "val" {
			valIter = "val"
			kv[1] = "Obj." + kv[1]
		} else {
			m.gen.Fail(*message.Name, "invalid key in uses_org check spec, expected \"key\" or \"val\", but was ", kv[0])
		}
		kvChecks = append(kvChecks, kv)
	}
	m.P("c.Mux.Lock()")
	m.P("defer c.Mux.Unlock()")
	m.P("for ", keyIter, ", ", valIter, " := range c.Objs {")
	for _, kv := range kvChecks {
		m.P("if ", kv[0], ".", kv[1], " == org { return true }")
	}
	m.P("}")
	m.P("return false")
	m.P("}")
}

func (m *mex) generateService(file *generator.FileDescriptor, service *descriptor.ServiceDescriptorProto) {
	if len(service.Method) != 0 {
		if gensupport.GetInternalApi(service) {
			return
		}
		for _, method := range service.Method {
			m.generateMethod(file, service, method)
		}
	}
}

func (m *mex) generateMethod(file *generator.FileDescriptor, service *descriptor.ServiceDescriptorProto, method *descriptor.MethodDescriptorProto) {
	in := gensupport.GetDesc(m.gen, method.GetInputType())
	if !gensupport.IsShow(method) {
		m.P("func (m *", *in.DescriptorProto.Name, ") IsValidArgsFor", *method.Name, "() error {")
		m.getInvalidMethodFields([]string{""}, false, in, method)
		m.P("return nil")
		m.P("}")
		m.P("")
	}
}

func (m *mex) checkDeletePrepares() {
	msgs := []string{}
	for _, ref := range m.refData.RefTos {
		if !ref.To.GenerateCud {
			continue
		}
		// refTo object must have delete_prepare boolean field.
		hierName := gensupport.GetDeletePrepareField(m.gen, ref.To.TypeDesc)
		if hierName == "" {
			refsBy := []string{}
			for _, by := range ref.Bys {
				refsBy = append(refsBy, by.By.Type+"."+by.Field.HierName)
			}
			errMsg := fmt.Sprintf("%s requires bool field %s, which is needed for safe deletes due to references from %s.", ref.To.Type, gensupport.DeletePrepareName, strings.Join(refsBy, ", "))
			msgs = append(msgs, errMsg)
		} else {
			m.deletePrepareFields[ref.To.Type] = hierName
		}
	}
	help := fmt.Sprintf("Deletes must first use ApplySTMWait to set %s to true (which waits until local cache is updated, ensuring other changes have updated the caches), then search via caches to make sure no references to it exist, then delete.", gensupport.DeletePrepareName)
	if len(msgs) > 0 {
		msgs = append(msgs, help)
		m.gen.Fail("\n" + strings.Join(msgs, "\n"))
	}
}

func (m *mex) generateGetReferences() {
	if len(m.refData.RefBys) == 0 {
		return
	}
	m.P()
	m.P("// References generated from the refers_to and tracks_refs_by protogen options")
	m.P("func GetReferencesMap() map[string][]string {")
	m.P("refs := make(map[string][]string)")
	refMap := make(map[string]map[string]struct{})
	for obj, refByGroup := range m.refData.RefBys {
		refs := map[string]struct{}{}
		for _, to := range refByGroup.Tos {
			refs[to.To.Type] = struct{}{}
		}
		refMap[obj] = refs
	}
	for obj, tracker := range m.refData.Trackers {
		refs := map[string]struct{}{}
		for _, by := range tracker.Bys {
			refs[by.By.Type] = struct{}{}
		}
		refMap[obj] = refs
	}
	objs := []string{}
	for obj, _ := range refMap {
		objs = append(objs, obj)
	}
	sort.Strings(objs)
	for _, obj := range objs {
		refs := refMap[obj]
		strs := []string{}
		for ref, _ := range refs {
			strs = append(strs, "\""+ref+"\"")
		}
		sort.Strings(strs)
		m.P(fmt.Sprintf("refs[%q] = []string{%s}", obj, strings.Join(strs, ", ")))
	}
	m.P("return refs")
	m.P("}")
}

func GetGenerateMatches(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_GenerateMatches, false)
}

func GetGenerateCopyInFields(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_GenerateCopyInFields, true)
}

func GetGenerateCud(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_GenerateCud, false)
}

func GetGenerateCache(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_GenerateCache, false)
}

func GetGenerateWaitForState(message *descriptor.DescriptorProto) string {
	return gensupport.GetStringExtension(message.Options, protogen.E_GenerateWaitForState, "")
}

func GetGenerateLookupBySublist(message *descriptor.DescriptorProto) string {
	return gensupport.GetStringExtension(message.Options, protogen.E_GenerateLookupBySublist, "")
}

func GetGenerateLookupBySubfield(message *descriptor.DescriptorProto) string {
	return gensupport.GetStringExtension(message.Options, protogen.E_GenerateLookupBySubfield, "")
}

func GetNotifyCache(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_NotifyCache, false)
}

func GetNotifyMessage(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_NotifyMessage, false)
}

func GetNotifyFlush(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_NotifyFlush, false)
}

func GetObjKey(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_ObjKey, false)
}

func GetGenerateStreamKey(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_GenerateStreamKey, false)
}

func GetUsesOrg(message *descriptor.DescriptorProto) string {
	return gensupport.GetStringExtension(message.Options, protogen.E_UsesOrg, "")
}

func GetCopyInAllFields(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_CopyInAllFields, false)
}

func GetSingularData(message *descriptor.DescriptorProto) bool {
	return proto.GetBoolExtension(message.Options, protogen.E_SingularData, false)
}

func GetBackend(field *descriptor.FieldDescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, protogen.E_Backend, false)
}

func GetHideTag(field *descriptor.FieldDescriptorProto) string {
	return gensupport.GetStringExtension(field.Options, protogen.E_Hidetag, "")
}

func GetKeyTag(field *descriptor.FieldDescriptorProto) string {
	return gensupport.GetStringExtension(field.Options, protogen.E_Keytag, "")
}

func GetSkipKeyTagConflictCheck(field *descriptor.FieldDescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, protogen.E_SkipKeytagConflictCheck, false)
}

func GetVersionHashOpt(enum *descriptor.EnumDescriptorProto) bool {
	return proto.GetBoolExtension(enum.Options, protogen.E_VersionHash, false)
}

func GetVersionHashSalt(enum *descriptor.EnumDescriptorProto) string {
	return gensupport.GetStringExtension(enum.Options, protogen.E_VersionHashSalt, "")
}

func GetUpgradeFunc(enumVal *descriptor.EnumValueDescriptorProto) string {
	return gensupport.GetStringExtension(enumVal.Options, protogen.E_UpgradeFunc, "")
}

func GetRedisOnly(field *descriptor.FieldDescriptorProto) bool {
	return proto.GetBoolExtension(field.Options, protogen.E_RedisOnly, false)
}

func GetParentObjName(message *descriptor.DescriptorProto) string {
	return gensupport.GetStringExtension(message.Options, protogen.E_ParentObjName, "")
}
