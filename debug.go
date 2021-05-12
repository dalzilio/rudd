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

// +build debug

package rudd

import (
	"log"
	"os"
)

const _DEBUG bool = true
const _LOGLEVEL int = 1

// ******************************************************************************************************

func init() {
	log.SetOutput(os.Stdout)
}

// ******************************************************************************************************

func (b *buddy) logTable() {
	if b.error != nil {
		log.Printf("ERROR: %s\n", b.error)
	}
	for k, n := range b.nodes {
		hash := b.ptrhash(k)
		switch {
		case n.refcou == _MAXREFCOUNT:
			log.Printf("%-3d ( %-3d ,  %-3d ,  %-3d) # %-3d  |hash:  %-3d  |next:  %-3d | +\n", k, n.level, n.low, n.high, hash, n.hash, n.next)
		case n.refcou == 0:
			log.Printf("%-3d ( %-3d ,  %-3d ,  %-3d) # %-3d  |hash:  %-3d  |next:  %-3d |\n", k, n.level, n.low, n.high, hash, n.hash, n.next)
		default:
			log.Printf("%-3d ( %-3d ,  %-3d ,  %-3d) # %-3d  |hash:  %-3d  |next:  %-3d | %d\n", k, n.level, n.low, n.high, hash, n.hash, n.next, n.refcou)
		}
	}
}
