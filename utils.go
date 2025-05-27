// Copyright 2020 - 2022 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	copyright = `// Plasmatic MX Message Parsing Library
// https://github.com/GoPlasmatic/MXMessage
//
// Copyright (c) 2025 Plasmatic
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// You may obtain a copy of this library at
// https://github.com/GoPlasmatic/MXMessage`
	matchFirstCap  = regexp.MustCompile("([A-Z])([A-Z][a-z])")
	matchAllCap    = regexp.MustCompile("([a-z0-9])([A-Z])")
	fieldNameCount map[string]int
)

// ToSnakeCase converts the provided string to snake_case.
func ToSnakeCase(input string) string {
	output := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	output = matchAllCap.ReplaceAllString(output, "${1}_${2}")
	output = strings.ReplaceAll(output, "-", "_")
	return strings.ToLower(output)
}

// GetFileList get a list of file by given path.
func GetFileList(path string) (files []string, err error) {
	var fi os.FileInfo
	fi, err = os.Stat(path)
	if err != nil {
		return
	}
	if fi.IsDir() {
		err = filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
			files = append(files, fp)
			return nil
		})
		if err != nil {
			return
		}
	}
	files = append(files, path)
	return
}

// PrepareOutputDir provide a method to create the output directory by given
// path.
func PrepareOutputDir(path string) error {
	if path == "" {
		return nil
	}
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

// BuildInTypes defines the correspondence between Go, TypeScript, C, Java,
// Rust languages and data types in XSD.
// https://www.w3.org/TR/xmlschema-2/#datatype
var BuildInTypes = map[string][]string{
	"anyType":            {"string", "string", "char", "String", "String"},
	"ENTITIES":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>"},
	"ENTITY":             {"string", "string", "char", "String", "String"},
	"ID":                 {"string", "string", "char", "String", "String"},
	"IDREF":              {"string", "string", "char", "String", "String"},
	"IDREFS":             {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>"},
	"NCName":             {"string", "string", "char", "String", "String"},
	"NMTOKEN":            {"string", "string", "char", "String", "String"},
	"NMTOKENS":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>"},
	"NOTATION":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>"},
	"Name":               {"string", "string", "char", "String", "String"},
	"QName":              {"xml.Name", "any", "char", "String", "String"},
	"anyURI":             {"string", "string", "char", "QName", "String"},
	"base64Binary":       {"string", "Uint8Array", "char[]", "List<Byte>", "String"},
	"boolean":            {"bool", "boolean", "bool", "Boolean", "bool"},
	"byte":               {"int8", "any", "char[]", "Byte", "u8"},
	"date":               {"string", "string", "char", "String", "String"},
	"dateTime":           {"string", "string", "char", "String", "String"},
	"decimal":            {"float64", "number", "float", "Float", "f64"},
	"double":             {"float64", "number", "float", "Float", "f64"},
	"duration":           {"string", "string", "char", "String", "String"},
	"float":              {"float32", "number", "float", "Float", "f64"},
	"gDay":               {"string", "string", "char", "String", "String"},
	"gMonth":             {"string", "string", "char", "String", "String"},
	"gMonthDay":          {"string", "string", "char", "String", "String"},
	"gYear":              {"string", "string", "char", "String", "String"},
	"gYearMonth":         {"string", "string", "char", "String", "String"},
	"hexBinary":          {"string", "Uint8Array", "char[]", "List<Byte>", "String"},
	"int":                {"int", "number", "int", "Integer", "i32"},
	"integer":            {"int", "number", "int", "Integer", "i32"},
	"language":           {"string", "string", "char", "String", "String"},
	"long":               {"int64", "number", "int", "Long", "i64"},
	"negativeInteger":    {"int", "number", "int", "Integer", "i32"},
	"nonNegativeInteger": {"int", "number", "int", "Integer", "u32"},
	"normalizedString":   {"string", "string", "char", "String", "String"},
	"nonPositiveInteger": {"int", "number", "int", "Integer", "i32"},
	"positiveInteger":    {"int", "number", "int", "Integer", "u32"},
	"short":              {"int16", "number", "int", "Integer", "i16"},
	"string":             {"string", "string", "char", "String", "String"},
	"time":               {"time.Time", "string", "char", "String", "String"},
	"token":              {"string", "string", "char", "String", "String"},
	"unsignedByte":       {"uint8", "any", "char", "Byte", "u8"},
	"unsignedInt":        {"uint32", "number", "unsigned int", "Integer", "u32"},
	"unsignedLong":       {"uint64", "number", "unsigned int", "Long", "u64"},
	"unsignedShort":      {"uint16", "number", "unsigned int", "Short", "u16"},
	"xml:lang":           {"string", "string", "char", "String", "String"},
	"xml:space":          {"string", "string", "char", "String", "String"},
	"xml:base":           {"string", "string", "char", "String", "String"},
	"xml:id":             {"string", "string", "char", "String", "String"},
}

func getBuildInTypeByLang(value, lang string) (buildType string, ok bool) {
	var supportLang = map[string]int{
		"Go":         0,
		"TypeScript": 1,
		"C":          2,
		"Java":       3,
		"Rust":       4,
	}
	var buildInTypes []string
	if buildInTypes, ok = BuildInTypes[value]; !ok {
		return
	}
	buildType = buildInTypes[supportLang[lang]]
	return
}

func getBasefromSimpleType(name string, XSDSchema []interface{}) string {
	for _, ele := range XSDSchema {
		switch v := ele.(type) {
		case *SimpleType:
			if !(v.Restriction.Pattern == nil && len(v.Restriction.Enum) == 0 && !v.Restriction.hasMinLength && !v.Restriction.hasMaxLength) && v.Name == name {
				return v.Name
			} else if !v.List && !v.Union && v.Name == name {
				return v.Base
			}
		case *Attribute:
			if v.Name == name {
				return v.Type
			}
		case *Element:
			if v.Name == name {
				return v.Type
			}
		}
	}
	return name
}

func getRefSimpleType(name string, XSDSchema []interface{}) *SimpleType {
	for _, ele := range XSDSchema {
		switch v := ele.(type) {
		case *SimpleType:
			if !v.List && !v.Union && v.Name == name {
				return v
			}
		}
	}
	return nil
}

func getNSPrefix(str string) (ns string) {
	split := strings.Split(str, ":")
	if len(split) == 2 {
		ns = split[0]
		return
	}
	return
}

func trimNSPrefix(str string) (name string) {
	split := strings.Split(str, ":")
	if len(split) == 2 {
		name = split[1]
		return
	}
	name = str
	return
}

// MakeFirstUpperCase make the first letter of a string uppercase.
func MakeFirstUpperCase(s string) string {
	return ToTitle(s)
}

func ToTitle(val string) string {
	var buf strings.Builder
	buf.Grow(utf8.UTFMax * len(val))

	for i, rune := range val {
		if i == 0 {
			rune = unicode.ToUpper(rune)
		}
		buf.WriteRune(rune)
	}

	return buf.String()
}

// callFuncByName calls the no error or only error return function with
// reflect by given receiver, name and parameters.
func callFuncByName(receiver interface{}, name string, params []reflect.Value) (err error) {
	function := reflect.ValueOf(receiver).MethodByName(name)
	if function.IsValid() {
		rt := function.Call(params)
		if len(rt) == 0 {
			return
		}
		if !rt[0].IsNil() {
			err = rt[0].Interface().(error)
			return
		}
	}
	return
}

// isValidUrl tests a string to determine if it is a well-structured url or
// not.
func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func fetchSchema(URL string) ([]byte, error) {
	var body []byte
	var client http.Client
	var err error
	resp, err := client.Get(URL)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return body, err
		}
	}
	return body, err
}

func genFieldComment(name, doc, prefix string) string {
	docReplacer := strings.NewReplacer("\n", fmt.Sprintf("\r\n%s ", prefix), "\t", "")
	if doc == "" {
		return fmt.Sprintf("\r\n%s %s ...\r\n", prefix, name)
	}
	return fmt.Sprintf("\r\n%s %s: %s\r\n", prefix, name, docReplacer.Replace(doc))
}

type kvPair struct {
	key   string
	value string
}

// kvPairList adapted from Andrew Gerrand for a similar problem (sorted map): https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
type kvPairList []kvPair

func (k kvPairList) Len() int { return len(k) }

func (k kvPairList) Less(i, j int) bool {
	return k[i].value < k[j].value || (k[i].value == k[j].value && strings.Compare(k[i].key, k[j].key) > 0)
}

func (k kvPairList) Swap(i, j int) { k[i], k[j] = k[j], k[i] }

func toSortedPairs(toSort map[string]string) kvPairList {
	pl := make(kvPairList, 0, len(toSort))
	for k, v := range toSort {
		pl = append(pl, kvPair{k, v})
	}

	sort.Sort(pl)
	return pl
}
