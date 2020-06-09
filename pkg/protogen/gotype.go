// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// From cmd/protoc-gen-go/internal_gengo/main.go.

package protogen

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func FieldGoType(field protoreflect.FieldDescriptor) (string, bool) {
	var goType string
	var pointer bool

	if field.IsWeak() {
		return "struct{}", false
	}

	pointer = field.HasPresence()
	switch field.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		/* TODO(jpeach)
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
		*/
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		/* TODO(jpeach)
		goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false // pointer captured as part of the type
		*/
	}

	switch {
	case field.IsList():
		return "[]" + goType, false
	case field.IsMap():
		/* TODO(jpeach)
		keyType, _ := FieldGoType(field.Message.Fields[0])
		valType, _ := FieldGoType(field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
		*/
	}

	return goType, pointer
}
