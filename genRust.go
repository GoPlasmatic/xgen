// Copyright 2020 - 2024 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

var (
	rustBuildinType = map[string]bool{
		"i8":          true,
		"i16":         true,
		"i32":         true,
		"i64":         true,
		"i128":        true,
		"isize":       true,
		"u8":          true,
		"u16":         true,
		"u32":         true,
		"u64":         true,
		"u128":        true,
		"usize":       true,
		"f32":         true,
		"f64":         true,
		"Vec<char>":   true,
		"Vec<String>": true,
		"Vec<u8>":     true,
		"bool":        true,
		"char":        true,
		"String":      true,
	}
	rustKeywords = map[string]bool{
		"as":       true,
		"break":    true,
		"const":    true,
		"continue": true,
		"crate":    true,
		"dyn":      true,
		"else":     true,
		"enum":     true,
		"extern":   true,
		"false":    true,
		"fn":       true,
		"for":      true,
		"if":       true,
		"impl":     true,
		"in":       true,
		"let":      true,
		"loop":     true,
		"match":    true,
		"mod":      true,
		"move":     true,
		"mut":      true,
		"pub":      true,
		"ref":      true,
		"return":   true,
		"Self":     true,
		"self":     true,
		"static":   true,
		"struct":   true,
		"super":    true,
		"trait":    true,
		"true":     true,
		"type":     true,
		"unsafe":   true,
		"use":      true,
		"where":    true,
		"while":    true,
		"abstract": true,
		"async":    true,
		"await":    true,
		"become":   true,
		"box":      true,
		"do":       true,
		"final":    true,
		"macro":    true,
		"override": true,
		"priv":     true,
		"try":      true,
		"typeof":   true,
		"unsized":  true,
		"virtual":  true,
		"yield":    true,
	}
	commonDerives = `#[derive(Debug, Default, Serialize, Deserialize, Clone, PartialEq)]
`
)

// GenRust generate Go programming language source code for XML schema
// definition files.
func (gen *CodeGenerator) GenRust() error {
	fieldNameCount = make(map[string]int)
	for _, ele := range gen.ProtoTree {
		if ele == nil {
			continue
		}
		funcName := fmt.Sprintf("Rust%s", reflect.TypeOf(ele).String()[6:])
		callFuncByName(gen, funcName, []reflect.Value{reflect.ValueOf(ele)})
	}
	f, err := os.Create(gen.FileWithExtension(".rs"))
	if err != nil {
		return err
	}
	defer f.Close()
	var imports = `use crate::parse_result::{ErrorCollector, ParserConfig};
use crate::validation::{Validate, helpers};
use serde::{Deserialize, Serialize};`
	source := []byte(fmt.Sprintf("%s\n%s\n%s", copyright, imports, gen.Field))
	f.Write(source)
	return err
}

// genRustFieldName generate struct field name for Rust code.
func genRustFieldName(name string) (fieldName string) {
	for _, str := range strings.Split(name, ":") {
		fieldName += MakeFirstUpperCase(str)
	}
	var tmp string
	for _, str := range strings.Split(fieldName, ".") {
		tmp += MakeFirstUpperCase(str)
	}
	fieldName = tmp
	fieldName = ToSnakeCase(strings.Replace(fieldName, "-", "", -1))
	if _, ok := rustKeywords[fieldName]; ok {
		fieldName += "_attr"
	}
	return
}

// genRustStructName generate struct name for Rust code.
func genRustStructName(name string, unique bool) (structName string) {
	for _, str := range strings.Split(name, ":") {
		structName += MakeFirstUpperCase(str)
	}
	var tmp string
	for _, str := range strings.Split(structName, ".") {
		tmp += MakeFirstUpperCase(str)
	}
	structName = tmp
	structName = strings.NewReplacer("-", "", "_", "").Replace(structName)
	if unique {
		fieldNameCount[structName]++
		if count := fieldNameCount[structName]; count != 1 {
			structName = fmt.Sprintf("%s%d", structName, count)
		}
	}
	return
}

// genRustFieldType generate struct field type for Rust code.
func genRustFieldType(name string) string {
	if _, ok := rustBuildinType[name]; ok {
		return name
	}
	fieldType := genRustStructName(name, false)
	if fieldType != "" {
		return fieldType
	}
	return "char"
}

func escapeRustString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// Helper function to generate validation code with error collection for string types
func genStringValidationWithPath(fieldName string, xmlName string, restriction *Restriction, optional bool, plural bool) string {
	if restriction == nil {
		return ""
	}

	validations := ""

	// Generate length validation
	if restriction.hasMinLength || restriction.hasMaxLength {
		minStr := "None"
		maxStr := "None"
		if restriction.hasMinLength {
			minStr = fmt.Sprintf("Some(%d)", restriction.MinLength)
		}
		if restriction.hasMaxLength {
			maxStr = fmt.Sprintf("Some(%d)", restriction.MaxLength)
		}

		if plural {
			if optional {
				validations += fmt.Sprintf("if let Some(ref vec) = self.%s {\n", fieldName)
				validations += fmt.Sprintf("\tfor item in vec {\n")
				validations += fmt.Sprintf("\t\thelpers::validate_length(\n")
				validations += fmt.Sprintf("\t\t\titem,\n")
				validations += fmt.Sprintf("\t\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t\t%s,\n", minStr)
				validations += fmt.Sprintf("\t\t\t%s,\n", maxStr)
				validations += fmt.Sprintf("\t\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\t\tconfig,\n")
				validations += fmt.Sprintf("\t\t\tcollector,\n")
				validations += fmt.Sprintf("\t\t);\n")
				validations += fmt.Sprintf("\t}\n")
				validations += fmt.Sprintf("}\n")
			} else {
				validations += fmt.Sprintf("for item in &self.%s {\n", fieldName)
				validations += fmt.Sprintf("\thelpers::validate_length(\n")
				validations += fmt.Sprintf("\t\titem,\n")
				validations += fmt.Sprintf("\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t%s,\n", minStr)
				validations += fmt.Sprintf("\t\t%s,\n", maxStr)
				validations += fmt.Sprintf("\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\tconfig,\n")
				validations += fmt.Sprintf("\t\tcollector,\n")
				validations += fmt.Sprintf("\t);\n")
				validations += fmt.Sprintf("}\n")
			}
		} else {
			if optional {
				validations += fmt.Sprintf("if let Some(ref val) = self.%s {\n", fieldName)
				validations += fmt.Sprintf("\thelpers::validate_length(\n")
				validations += fmt.Sprintf("\t\tval,\n")
				validations += fmt.Sprintf("\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t%s,\n", minStr)
				validations += fmt.Sprintf("\t\t%s,\n", maxStr)
				validations += fmt.Sprintf("\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\tconfig,\n")
				validations += fmt.Sprintf("\t\tcollector,\n")
				validations += fmt.Sprintf("\t);\n")
				validations += fmt.Sprintf("}\n")
			} else {
				validations += fmt.Sprintf("helpers::validate_length(\n")
				validations += fmt.Sprintf("\t&self.%s,\n", fieldName)
				validations += fmt.Sprintf("\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t%s,\n", minStr)
				validations += fmt.Sprintf("\t%s,\n", maxStr)
				validations += fmt.Sprintf("\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\tconfig,\n")
				validations += fmt.Sprintf("\tcollector,\n")
				validations += fmt.Sprintf(");\n")
			}
		}
	}

	// Generate pattern validation
	if restriction.Pattern != nil {
		patternStr := escapeRustString(restriction.Pattern.String())
		if plural {
			if optional {
				validations += fmt.Sprintf("if let Some(ref vec) = self.%s {\n", fieldName)
				validations += fmt.Sprintf("\tfor item in vec {\n")
				validations += fmt.Sprintf("\t\thelpers::validate_pattern(\n")
				validations += fmt.Sprintf("\t\t\titem,\n")
				validations += fmt.Sprintf("\t\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t\t\"%s\",\n", patternStr)
				validations += fmt.Sprintf("\t\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\t\tconfig,\n")
				validations += fmt.Sprintf("\t\t\tcollector,\n")
				validations += fmt.Sprintf("\t\t);\n")
				validations += fmt.Sprintf("\t}\n")
				validations += fmt.Sprintf("}\n")
			} else {
				validations += fmt.Sprintf("for item in &self.%s {\n", fieldName)
				validations += fmt.Sprintf("\thelpers::validate_pattern(\n")
				validations += fmt.Sprintf("\t\titem,\n")
				validations += fmt.Sprintf("\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t\"%s\",\n", patternStr)
				validations += fmt.Sprintf("\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\tconfig,\n")
				validations += fmt.Sprintf("\t\tcollector,\n")
				validations += fmt.Sprintf("\t);\n")
				validations += fmt.Sprintf("}\n")
			}
		} else {
			if optional {
				validations += fmt.Sprintf("if let Some(ref val) = self.%s {\n", fieldName)
				validations += fmt.Sprintf("\thelpers::validate_pattern(\n")
				validations += fmt.Sprintf("\t\tval,\n")
				validations += fmt.Sprintf("\t\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\t\"%s\",\n", patternStr)
				validations += fmt.Sprintf("\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\t\tconfig,\n")
				validations += fmt.Sprintf("\t\tcollector,\n")
				validations += fmt.Sprintf("\t);\n")
				validations += fmt.Sprintf("}\n")
			} else {
				validations += fmt.Sprintf("helpers::validate_pattern(\n")
				validations += fmt.Sprintf("\t&self.%s,\n", fieldName)
				validations += fmt.Sprintf("\t\"%s\",\n", xmlName)
				validations += fmt.Sprintf("\t\"%s\",\n", patternStr)
				validations += fmt.Sprintf("\t&helpers::child_path(path, \"%s\"),\n", xmlName)
				validations += fmt.Sprintf("\tconfig,\n")
				validations += fmt.Sprintf("\tcollector,\n")
				validations += fmt.Sprintf(");\n")
			}
		}
	}

	return validations
}

// Helper function to generate validation for custom types with error collection
func genCustomTypeValidationWithPath(fieldName string, xmlName string, fieldType string, optional bool, plural bool) string {
	// Only call validate_with_path on custom types
	if fieldType == "String" || fieldType == "i32" || fieldType == "f64" || fieldType == "bool" {
		return ""
	}

	validations := ""

	if plural {
		if optional {
			validations += fmt.Sprintf("if let Some(ref vec) = self.%s {\n", fieldName)
			validations += fmt.Sprintf("\tif config.validate_optional_fields {\n")
			validations += fmt.Sprintf("\t\tfor item in vec {\n")
			validations += fmt.Sprintf("\t\t\titem.validate(\n")
			validations += fmt.Sprintf("\t\t\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
			validations += fmt.Sprintf("\t\t\t\tconfig,\n")
			validations += fmt.Sprintf("\t\t\t\tcollector,\n")
			validations += fmt.Sprintf("\t\t\t);\n")
			validations += fmt.Sprintf("\t\t}\n")
			validations += fmt.Sprintf("\t}\n")
			validations += fmt.Sprintf("}\n")
		} else {
			validations += fmt.Sprintf("for item in &self.%s {\n", fieldName)
			validations += fmt.Sprintf("\titem.validate(\n")
			validations += fmt.Sprintf("\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
			validations += fmt.Sprintf("\t\tconfig,\n")
			validations += fmt.Sprintf("\t\tcollector,\n")
			validations += fmt.Sprintf("\t);\n")
			validations += fmt.Sprintf("}\n")
		}
	} else {
		if optional {
			validations += fmt.Sprintf("if let Some(ref val) = self.%s {\n", fieldName)
			validations += fmt.Sprintf("\tif config.validate_optional_fields {\n")
			validations += fmt.Sprintf("\t\tval.validate(\n")
			validations += fmt.Sprintf("\t\t\t&helpers::child_path(path, \"%s\"),\n", xmlName)
			validations += fmt.Sprintf("\t\t\tconfig,\n")
			validations += fmt.Sprintf("\t\t\tcollector,\n")
			validations += fmt.Sprintf("\t\t);\n")
			validations += fmt.Sprintf("\t}\n")
			validations += fmt.Sprintf("}\n")
		} else {
			validations += fmt.Sprintf("self.%s.validate(\n", fieldName)
			validations += fmt.Sprintf("\t&helpers::child_path(path, \"%s\"),\n", xmlName)
			validations += fmt.Sprintf("\tconfig,\n")
			validations += fmt.Sprintf("\tcollector,\n")
			validations += fmt.Sprintf(");\n")
		}
	}

	return validations
}

// OldValidationResult holds the separated regex setup and validation body code
type OldValidationResult struct {
	RegexSetup string // Code to create regex (should be outside loops)
	LoopBody   string // Validation code (can be inside loops)
}

// Old validation code generation for backward compatibility validate() method
// Returns regex setup code separately to avoid compiling regex inside loops
func getOldValidationCode(variable string, fieldName string, fieldType string, restriction *Restriction) OldValidationResult {
	result := OldValidationResult{}
	validations := ""

	// Handle minLength and maxLength for string types
	if restriction.hasMinLength {
		validations += fmt.Sprintf("if %s.chars().count() < %d {\n", variable, restriction.MinLength)
		validations += fmt.Sprintf("\treturn Err(ValidationError::new(1001, \"%s is shorter than the minimum length of %d\".to_string()));\n", fieldName, restriction.MinLength)
		validations += "}\n"
	}
	if restriction.hasMaxLength {
		validations += fmt.Sprintf("if %s.chars().count() > %d {\n", variable, restriction.MaxLength)
		validations += fmt.Sprintf("\treturn Err(ValidationError::new(1002, \"%s exceeds the maximum length of %d\".to_string()));\n", fieldName, restriction.MaxLength)
		validations += "}\n"
	}

	// Handle minInclusive and maxInclusive for numeric types
	if restriction.hasMin {
		v := variable
		if v == "val" || v == "item" {
			v = "*" + v
		}
		validations += fmt.Sprintf("if %s < %f {\n", v, restriction.Min)
		validations += fmt.Sprintf("\treturn Err(ValidationError::new(1003, \"%s is less than the minimum value of %f\".to_string()));\n", fieldName, restriction.Min)
		validations += "}\n"
	}
	if restriction.hasMax {
		v := variable
		if v == "val" || v == "item" {
			v = "*" + v
		}
		validations += fmt.Sprintf("if %s > %f {\n", v, restriction.Max)
		validations += fmt.Sprintf("\treturn Err(ValidationError::new(1004, \"%s exceeds the maximum value of %f\".to_string()));\n", fieldName, restriction.Max)
		validations += "}\n"
	}

	// Handle pattern constraints for string types
	// Regex creation is returned separately to be placed outside loops
	if restriction.Pattern != nil && fieldType == "String" {
		patternStr := escapeRustString(restriction.Pattern.String())
		result.RegexSetup = fmt.Sprintf("let pattern = Regex::new(\"%s\").unwrap();\n", patternStr)
		if variable == "val" {
			validations += fmt.Sprintf("if !pattern.is_match(%s) {\n", variable)
		} else {
			validations += fmt.Sprintf("if !pattern.is_match(&%s) {\n", variable)
		}
		validations += fmt.Sprintf("\treturn Err(ValidationError::new(1005, \"%s does not match the required pattern\".to_string()));\n", fieldName)
		validations += "}\n"
	}

	if len(validations) > 0 {
		i := strings.LastIndex(validations, "\n")
		validations = validations[:i]
	}

	result.LoopBody = validations
	return result
}

// Helper function to generate old validation for built-in types
func genBuiltInOldValidation(fieldName string, fieldType string, restriction *Restriction, plural bool, optional bool) string {
	validations := ""

	// Handle plural (Vec) case for built-in types
	if plural {
		result := getOldValidationCode("item", fieldName, fieldType, restriction)
		if len(result.LoopBody) > 0 || len(result.RegexSetup) > 0 {
			if optional {
				// Handle Option<Vec<T>> for built-in types
				// Place regex setup outside the loop but inside the if-let
				validations += fmt.Sprintf("if let Some(ref vec) = self.%s {\n", fieldName)
				if len(result.RegexSetup) > 0 {
					validations += fmt.Sprintf("\t%s", result.RegexSetup)
				}
				validations += fmt.Sprintf("\tfor item in vec {\n\t\t%s\n\t}\n}\n", strings.ReplaceAll(result.LoopBody, "\n", "\n\t\t"))
			} else {
				// Handle Vec<T> for built-in types
				// Place regex setup outside the loop
				if len(result.RegexSetup) > 0 {
					validations += result.RegexSetup
				}
				validations += fmt.Sprintf("for item in &self.%s {\n\t%s\n}\n", fieldName, strings.ReplaceAll(result.LoopBody, "\n", "\n\t"))
			}
		}
	} else {
		// Handle Option<T> case
		if optional {
			result := getOldValidationCode("val", fieldName, fieldType, restriction)
			if len(result.LoopBody) > 0 || len(result.RegexSetup) > 0 {
				validations += fmt.Sprintf("if let Some(ref val) = self.%s {\n", fieldName)
				if len(result.RegexSetup) > 0 {
					validations += fmt.Sprintf("\t%s", result.RegexSetup)
				}
				if len(result.LoopBody) > 0 {
					validations += fmt.Sprintf("\t%s\n", strings.ReplaceAll(result.LoopBody, "\n", "\n\t"))
				}
				validations += "}\n"
			}
		} else {
			// Handle T case
			result := getOldValidationCode(fmt.Sprintf("self.%s", fieldName), fieldName, fieldType, restriction)
			if len(result.RegexSetup) > 0 {
				validations += result.RegexSetup
			}
			if len(result.LoopBody) > 0 {
				validations += result.LoopBody + "\n"
			}
		}
	}

	return validations
}

// Helper function to handle old validation for custom types
func genCustomTypeOldValidation(fieldName string, fieldType string, plural bool, optional bool) string {
	// Only call validate() on custom types, not on built-in types like String
	if fieldType == "String" || fieldType == "i32" || fieldType == "f64" || fieldType == "bool" {
		return "" // No validate() call for primitive types
	}

	if plural {
		// Handle Option<Vec<T>> for custom types
		if optional {
			return fmt.Sprintf("if let Some(ref vec) = self.%[1]s { for item in vec { item.validate()? } }\n", fieldName)
		}
		// Handle Vec<T> for custom types
		return fmt.Sprintf("for item in &self.%[1]s { item.validate()? }\n", fieldName)
	} else {
		// Handle Option<T> and T cases for custom types
		if optional {
			return fmt.Sprintf("if let Some(ref val) = self.%[1]s { val.validate()? }\n", fieldName)
		}
		return fmt.Sprintf("self.%s.validate()?;\n", fieldName)
	}
}

// Main function - returns field content, old validations, new validations, and xml name
func genRustFieldCode(name string, ftype string, plural bool, optional bool, restriction *Restriction, untagged bool, attibute bool) (string, string, string, string) {
	fieldName := genRustFieldName(name)
	fieldType := genRustFieldType(ftype)
	oldValidations := ""
	newValidations := ""
	
	// Generate old validation code for backward compatibility
	if isRustBuiltInType(ftype) && restriction != nil {
		oldValidations = genBuiltInOldValidation(fieldName, fieldType, restriction, plural, optional)
	} else {
		oldValidations = genCustomTypeOldValidation(fieldName, fieldType, plural, optional)
	}

	// Generate new validation code with error collection
	if isRustBuiltInType(ftype) && restriction != nil {
		newValidations = genStringValidationWithPath(fieldName, genRustFieldRename(name), restriction, optional, plural)
	} else {
		newValidations = genCustomTypeValidationWithPath(fieldName, genRustFieldRename(name), fieldType, optional, plural)
	}

	// Adjust field type for Vec and Option cases
	if plural {
		fieldType = "Vec<" + fieldType + ">"
	}
	if optional {
		fieldType = "Option<" + fieldType + ">"
	}

	rename := genRustFieldRename(name)
	if untagged {
		rename = "$value"
	}

	if attibute {
		rename = "@" + rename
	}

	content := fmt.Sprintf("\n#[serde(rename = \"%s\"", rename)
	if optional {
		content += ", skip_serializing_if = \"Option::is_none\""
	}
	content += fmt.Sprintf(")]\npub %s: %s,", genRustFieldName(name), fieldType)

	return content, oldValidations, newValidations, genRustFieldRename(name)
}

func genRustStructCode(name string, doc string, fieldContent string, oldValidations string, newValidations string, untagged bool) string {
	extraTags := ""
	if untagged {
		extraTags += "#[serde(transparent) ]\n"
	}

	content := fmt.Sprintf("\n%s%s%spub struct %s {%s\n}\n", genFieldComment(name, doc, "//"), commonDerives, extraTags, name, strings.ReplaceAll(fieldContent, "\n", "\n\t"))
	
	// Generate Validate trait implementation
	content += fmt.Sprintf("\nimpl Validate for %s {\n", name)
	if len(newValidations) > 0 {
		content += "\tfn validate(&self, path: &str, config: &ParserConfig, collector: &mut ErrorCollector) {\n"
		content += "\t\t" + strings.ReplaceAll(newValidations, "\n", "\n\t\t")
		content += "\t}\n}\n"
	} else {
		content += "\tfn validate(&self, _path: &str, _config: &ParserConfig, _collector: &mut ErrorCollector) {\n"
		content += "\t}\n}\n"
	}
	
	return content
}

func genRustEnumCode(name string, doc string, fieldContent string) string {
	content := fmt.Sprintf("\n%s%spub enum %s {\n\t#[default]\n", doc, commonDerives, name)
	content += fieldContent
	content += "}\n"
	
	// Generate Validate trait implementation for enum
	content += fmt.Sprintf("\nimpl Validate for %s {\n", name)
	content += "\tfn validate(&self, _path: &str, _config: &ParserConfig, _collector: &mut ErrorCollector) {\n"
	content += "\t\t// Enum validation is typically empty\n"
	content += "\t}\n}\n"
	
	return content
}

// RustSimpleType generates code for simple type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustSimpleType(v *SimpleType) {
	if len(v.Restriction.Enum) > 0 && v.Base == "String" {
		fieldContent := ""
		for _, enumValue := range v.Restriction.Enum {
			fieldContent += fmt.Sprintf("\t#[serde(rename = \"%s\")]\n\tCode%s,\n", enumValue, strings.Replace(strings.ToUpper(enumValue), ".", "", -1))
		}
		gen.StructAST[v.Name] = fieldContent
		enumName := genRustStructName(v.Name, true)
		gen.Field += genRustEnumCode(enumName, genFieldComment(v.Name, v.Doc, "//"), fieldContent)
		return
	}
}

// RustComplexType generates code for complex type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustComplexType(v *ComplexType) {
	var content, oldValidation, newValidation string
	for _, attrGroup := range v.AttributeGroup {
		fieldType := getBasefromSimpleType(trimNSPrefix(attrGroup.Ref), gen.ProtoTree)
		conts, oldValids, newValids, _ := genRustFieldCode(attrGroup.Name, fieldType, false, false, nil, false, false)
		content += conts
		oldValidation += oldValids
		newValidation += newValids
	}
	for _, attribute := range v.Attributes {
		// fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
		fieldType := "String"
		conts, oldValids, newValids, _ := genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, nil, false, true)
		content += conts
		oldValidation += oldValids
		newValidation += newValids
	}
	for _, group := range v.Groups {
		fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
		conts, oldValids, newValids, _ := genRustFieldCode(group.Name, fieldType, group.Plural, false, nil, false, false)
		content += conts
		oldValidation += oldValids
		newValidation += newValids
	}
	for _, element := range v.Elements {
		var r *Restriction
		fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
		simple := getRefSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
		if simple != nil && len(simple.Restriction.Enum) == 0 {
			fieldType = simple.Base
			r = &simple.Restriction
		} else {
			r = &element.Restriction
		}

		conts, oldValids, newValids, _ := genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, r, false, false)
		content += conts
		oldValidation += oldValids
		newValidation += newValids
	}
	if len(v.Base) > 0 {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
		if isRustBuiltInType(v.Base) {
			conts, oldValids, newValids, _ := genRustFieldCode("value", fieldType, false, false, nil, false, false)
			content += conts
			oldValidation += oldValids
			newValidation += newValids
		} else {
			fmt.Printf("\n\n%s\n", fieldType)
			fieldName := genRustFieldName(fieldType)
			// If the type is not a built-in one, add the base type as a nested field tagged with flatten
			content += fmt.Sprintf("\t#[serde(flatten)]\n\tpub %s: %s,\n", fieldName, fieldType)
		}
	}

	if _, ok := gen.StructAST[v.Name]; !ok {
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], oldValidation, newValidation, false)
	} else {
		fmt.Printf("%s\n", content)
	}
}

func isRustBuiltInType(typeName string) bool {
	_, builtIn := rustBuildinType[typeName]
	return builtIn
}

// RustGroup generates code for group XML schema in Rust language syntax.
func (gen *CodeGenerator) RustGroup(v *Group) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var content, oldValidation, newValidation string
		for _, element := range v.Elements {
			fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
			conts, oldValids, newValids, _ := genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, &element.Restriction, false, false)
			content += conts
			oldValidation += oldValids
			newValidation += newValids
		}
		for _, group := range v.Groups {
			fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
			conts, oldValids, newValids, _ := genRustFieldCode(group.Name, fieldType, group.Plural, false, nil, false, false)
			content += conts
			oldValidation += oldValids
			newValidation += newValids
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], oldValidation, newValidation, false)
	}
}

// RustAttributeGroup generates code for attribute group XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustAttributeGroup(v *AttributeGroup) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var content, oldValidation, newValidation string
		for _, attribute := range v.Attributes {
			fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
			conts, oldValids, newValids, _ := genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, &attribute.Restriction, false, false)
			content += conts
			oldValidation += oldValids
			newValidation += newValids
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], oldValidation, newValidation, false)
	}
}

// RustElement generates code for element XML schema in Rust language syntax.
func (gen *CodeGenerator) RustElement(v *Element) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		content, oldValidation, newValidation, _ := genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction, false, false)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name], oldValidation, newValidation, false)
	}
}

// RustAttribute generates code for attribute XML schema in Rust language syntax.
func (gen *CodeGenerator) RustAttribute(v *Attribute) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		content, oldValidation, newValidation, _ := genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction, false, false)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name], oldValidation, newValidation, false)
	}
}

// genRustStructName generate struct name for Rust code.
func genRustFieldRename(name string) string {
	if strings.Count(name, ":") > 0 {
		return strings.Split(name, ":")[1]
	} else {
		if name == "value" {
			name = "$value"
		}
		return name
	}
}
