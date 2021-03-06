/*
 * translate_fdb_options.go
 *
 * This source file is part of the FoundationDB open source project
 *
 * Copyright 2013-2018 Apple Inc. and the FoundationDB project authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// FoundationDB Go options translator

package main

import (
	"encoding/xml"
	"fmt"
	"go/doc"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Option struct {
	Name        string `xml:"name,attr"`
	Code        int    `xml:"code,attr"`
	ParamType   string `xml:"paramType,attr"`
	ParamDesc   string `xml:"paramDescription,attr"`
	Description string `xml:"description,attr"`
	Hidden      bool   `xml:"hidden,attr"`
}
type Scope struct {
	Name   string `xml:"name,attr"`
	Option []Option
}
type Options struct {
	Scope []Scope
}

func writeOptString(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param string) error {
	return o.setOpt(%d, []byte(param))
}
`, receiver, function, opt.Code)
}

func writeOptBytes(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param []byte) error {
	return o.setOpt(%d, param)
}
`, receiver, function, opt.Code)
}

func writeOptInt(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param int64) error {
	b, e := int64ToBytes(param)
	if e != nil {
		return e
	}
	return o.setOpt(%d, b)
}
`, receiver, function, opt.Code)
}

func writeOptNone(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s() error {
	return o.setOpt(%d, nil)
}
`, receiver, function, opt.Code)
}

func writeOpt(receiver string, opt Option) {
	function := "Set" + translateName(opt.Name)

	fmt.Println()

	if opt.Description != "" {
		fmt.Printf("// %s\n", opt.Description)
		if opt.ParamDesc != "" {
			fmt.Printf("//\n// Parameter: %s\n", opt.ParamDesc)
		}
	} else {
		fmt.Printf("// Not yet implemented.\n")
	}

	switch opt.ParamType {
	case "String":
		writeOptString(receiver, function, opt)
	case "Bytes":
		writeOptBytes(receiver, function, opt)
	case "Int":
		writeOptInt(receiver, function, opt)
	case "":
		writeOptNone(receiver, function, opt)
	default:
		log.Fatalf("Totally unexpected ParamType %s", opt.ParamType)
	}
}

func translateName(old string) string {
	return strings.Replace(strings.Title(strings.Replace(old, "_", " ", -1)), " ", "", -1)
}

func writeMutation(opt Option) {
	tname := translateName(opt.Name)
	fmt.Printf(`
// %s
func (t Transaction) %s(key KeyConvertible, param []byte) {
	t.atomicOp(key.FDBKey(), param, %d)
}
`, opt.Description, tname, opt.Code)
}

func writeEnum(scope Scope, opt Option, delta int) {
	fmt.Println()
	if opt.Description != "" {
		doc.ToText(os.Stdout, opt.Description, "\t// ", "", 73)
		// fmt.Printf("	// %s\n", opt.Description)
	}
	fmt.Printf("	%s %s = %d\n", scope.Name+translateName(opt.Name), scope.Name, opt.Code+delta)
}

func main() {
	var err error

	v := Options{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	err = xml.Unmarshal(data, &v)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(`/*
 * generated.go
 *
 * This source file is part of the FoundationDB open source project
 *
 * Copyright 2013-2018 Apple Inc. and the FoundationDB project authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// DO NOT EDIT THIS FILE BY HAND. This file was generated using
// translate_fdb_options.go, part of the FoundationDB repository, and a copy of
// the fdb.options file (installed as part of the FoundationDB client, typically
// found as /usr/include/foundationdb/fdb.options).

// To regenerate this file, from the top level of a FoundationDB repository
// checkout, run:
// $ go run bindings/go/src/_util/translate_fdb_options.go < fdbclient/vexillographer/fdb.options > bindings/go/src/fdb/generated.go

package fdb

import (
	"bytes"
	"encoding/binary"
)

func int64ToBytes(i int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if e := binary.Write(buf, binary.LittleEndian, i); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
`)

	for _, scope := range v.Scope {
		if strings.HasSuffix(scope.Name, "Option") {
			receiver := scope.Name + "s"

			for _, opt := range scope.Option {
				if !opt.Hidden {
					writeOpt(receiver, opt)
				}
			}
			continue
		}

		if scope.Name == "MutationType" {
			for _, opt := range scope.Option {
				if !opt.Hidden {
					writeMutation(opt)
				}
			}
			continue
		}

		// We really need the default StreamingMode (0) to be ITERATOR
		var d int
		if scope.Name == "StreamingMode" {
			d = 1
		}

		// ConflictRangeType shouldn't be exported
		if scope.Name == "ConflictRangeType" {
			scope.Name = "conflictRangeType"
		}

		fmt.Printf(`
type %s int

const (
`, scope.Name)
		for _, opt := range scope.Option {
			if !opt.Hidden {
				writeEnum(scope, opt, d)
			}
		}
		fmt.Println(")")
	}
}
