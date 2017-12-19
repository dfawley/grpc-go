/*
 *
 * Copyright 2017 gRPC authors.
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
 *
 */

// Package testlib contains a collection of utilities to aid in writing tests.
// These are not intended for production use.
package testlib

import (
	"reflect"
)

// Replace sets *src to val and returns a function that will restore *src
// to the value it contained upon calling this function.  If the types of *src
// and val do not match, a panic will occur.
func Replace(src, val interface{}) (restore func()) {
	srcVal := reflect.ValueOf(src)
	origVal := reflect.ValueOf(srcVal.Elem().Interface())
	srcVal.Elem().Set(reflect.ValueOf(val))
	return func() { srcVal.Elem().Set(origVal) }
}
