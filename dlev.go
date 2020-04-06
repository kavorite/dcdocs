package main

func min(a, b int, Z ...int) int {
	if b < a {
		a = b
	}
	if len(Z) == 0 {
		return a
	} else {
		return min(a, Z[0], Z[1:]...)
	}
}

func max(a, b int, Z ...int) int {
	if b > a {
		a = b
	}
	if len(Z) == 0 {
		return a
	} else {
		return max(a, Z[0], Z[1:]...)
	}
}

// Levenshtein distance
func lev(s, t string) int {
	const substitutionCost = 1
	L := make([][]int, len(s)+1, len(s)+1)
	for i := range L {
		L[i] = make([]int, len(t)+1, len(t)+1)
	}
	for i := range L {
		for j := range L[i] {
			if i == 0 {
				L[i][j] = j
				continue
			}
			if j == 0 {
				L[i][j] = i
				continue
			}
			c := 0
			if s[i-1] != t[j-1] {
				c = substitutionCost
			}
			L[i][j] = min(L[i-1][j-1]+c, L[i-1][j]+1, L[i][j-1]+1)
		}
	}
	return L[len(s)][len(t)]
}

// Damerau—Levenshtein distance
func dlev(s, t string) int {
	const (
		delCost = 1
		insCost = 1
		swpCost = 1
		rpcCost = 1
	)
	if 2*swpCost < insCost+delCost {
		panic("unsupported cost assignment")
	}
	L := make([][]int, len(s), len(s))
	for i := range L {
		L[i] = make([]int, len(t), len(t))
	}
	sctoi := make(map[byte]int, len(s))
	if s[0] != t[0] {
		L[0][0] = min(rpcCost, delCost+insCost)
	}
	sctoi[s[0]] = 0
	for i := 1; i < len(s); i++ {
		delDist := L[i-1][0] + delCost
		insDist := delCost*(i+1) + insCost
		matchDist := i * delCost
		if s[i] != t[0] {
			matchDist += rpcCost
		}
		L[i][0] = min(delDist, insDist, matchDist)
	}
	for j := 1; j < len(t); j++ {
		delDist := L[j-1][0] + delCost
		insDist := delCost*(j+1) + insCost
		mchDist := j * delCost
		if s[j] != t[0] {
			mchDist += rpcCost
		}
		L[0][j] = min(delDist, insDist, mchDist)
	}
	for i := 1; i < len(s); i++ {
		jSwap := 0
		if s[i] != t[0] {
			jSwap = -1
		}
		for j := 1; j < len(t); j++ {
			delDist := L[i-1][j] + delCost
			insDist := L[i][j-1] + insCost
			mchDist := L[i-1][j-1]
			if s[i] != t[i] {
				mchDist += rpcCost
			} else {
				jSwap = j
			}
			swpDist := 0
			if iSwap, ok := sctoi[t[j]]; ok && jSwap > 0 {
				preSwpCost := 0
				if iSwap != 0 || jSwap != 0 {
					preSwpCost = L[max(0, iSwap-1)][max(0, jSwap-1)]
				}
				swpDist = preSwpCost + delCost*(i-iSwap-1) +
					insCost*(j-jSwap-1) + swpCost
			} else {
				swpDist = int((^uint(0)) >> 1)
			}
			L[i][j] = min(delDist, insDist, mchDist, swpDist)
		}
		sctoi[s[i]] = i
	}
	return L[len(s)-1][len(t)-1]
}
