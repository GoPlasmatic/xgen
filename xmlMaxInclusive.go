// Copyright 2020 - 2024 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"encoding/xml"
	"strconv"
)

func (opt *Options) OnMaxInclusive(ele xml.StartElement, protoTree []interface{}) (err error) {
	for _, attr := range ele.Attr {
		if attr.Name.Local == "value" {
			if opt.SimpleType.Peek() != nil {
				opt.SimpleType.Peek().(*SimpleType).Restriction.Max, _ = strconv.ParseFloat(attr.Value, 64)
				opt.SimpleType.Peek().(*SimpleType).Restriction.hasMax = true
			}
		}
	}

	return
}

// EndMaxInclusive handles parsing event on the maxInclusive end elements.
// MaxInclusive specifies the upper bounds for numeric values (the value must
// be less than or equal to this value).
func (opt *Options) EndMaxInclusive(ele xml.EndElement, protoTree []interface{}) (err error) {
	if opt.SimpleType.Len() > 0 && opt.Element.Len() > 0 {
		if opt.Element.Peek().(*Element).Type, err = opt.GetValueType(opt.SimpleType.Pop().(*SimpleType).Base, opt.ProtoTree); err != nil {
			return
		}
		opt.CurrentEle = ""
	}
	return
}
