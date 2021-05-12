// Copyright 2021. Silvano DAL ZILIO.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package rudd

import (
	"fmt"
	"log"
)

// Error returns the error status of the BDD.
func (b *buddy) Error() string {
	if b.error == nil {
		return ""
	}
	return b.error.Error()
}

// Errored returns true if there was an error during a computation.
func (b *buddy) Errored() bool {
	return b.error != nil
}

func (b *buddy) seterror(format string, a ...interface{}) Node {
	if b.error != nil {
		format = format + "; " + b.Error()
		b.error = fmt.Errorf(format, a...)
		return nil
	}
	b.error = fmt.Errorf(format, a...)
	if _DEBUG {
		log.Println(b.error)
	}
	return nil
}

// check performs a sanity check prior to accessing a node and return eventual
// error code.
func (b *buddy) checkptr(n Node) error {
	switch {
	case n == nil:
		panic("uncaught error")
		// return b.error
	case (*n < 0) || (*n >= len(b.nodes)):
		b.seterror("Illegal acces to node %d", n)
		return b.error
	case (*n >= 2) && (b.nodes[*n].low == -1):
		b.seterror("Illegal acces to node %d", n)
		return b.error
	}
	return nil
}
