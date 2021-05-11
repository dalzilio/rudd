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

import "math/big"

// functions for Prime number calculations

func hasFactor(src int, n int) bool {
	if (src != n) && (src%n == 0) {
		return true
	}
	return false
}

func hasEasyFactors(src int) bool {
	return hasFactor(src, 3) || hasFactor(src, 5) || hasFactor(src, 7) || hasFactor(src, 11) || hasFactor(src, 13)
}

func bdd_prime_gte(src int) int {
	if src%2 == 0 {
		src++
	}

	for {
		if hasEasyFactors(src) {
			src = src + 2
			continue
		}
		// ProbablyPrime is 100% accurate for inputs less than 2⁶⁴.
		if big.NewInt(int64(src)).ProbablyPrime(0) {
			return src
		}
		src = src + 2
	}
}

func bdd_prime_lte(src int) int {
	if src == 0 {
		return 1
	}

	if src%2 == 0 {
		src--
	}

	for {
		if hasEasyFactors(src) {
			src = src - 2
			continue
		}
		// ProbablyPrime is 100% accurate for inputs less than 2⁶⁴.
		if big.NewInt(int64(src)).ProbablyPrime(0) {
			return src
		}
		src = src - 2
	}
}
